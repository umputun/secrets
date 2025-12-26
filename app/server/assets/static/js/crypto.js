// crypto.js - client-side AES-128-GCM encryption for paranoid mode
// uses Web Crypto API, requires HTTPS (or localhost)

'use strict';

// type bytes for content detection after decryption
const TYPE_TEXT = 0x00;
const TYPE_FILE = 0x01;

// base64url encoding/decoding (URL-safe, no padding)
function base64urlEncode(bytes) {
    const binString = Array.from(bytes, (b) => String.fromCodePoint(b)).join('');
    return btoa(binString).replace(/\+/g, '-').replace(/\//g, '_').replace(/=+$/, '');
}

function base64urlDecode(str) {
    // restore standard base64
    let base64 = str.replace(/-/g, '+').replace(/_/g, '/');
    // add padding if needed
    while (base64.length % 4) {
        base64 += '=';
    }
    const binString = atob(base64);
    return Uint8Array.from(binString, (c) => c.codePointAt(0));
}

// check if Web Crypto API is available (requires HTTPS or localhost)
function checkCryptoAvailable() {
    return typeof crypto !== 'undefined' && typeof crypto.subtle !== 'undefined';
}

// generate 128-bit random key, returns 22-char base64url string
async function generateKey() {
    const keyBytes = new Uint8Array(16); // 128 bits
    crypto.getRandomValues(keyBytes);
    return base64urlEncode(keyBytes);
}

// import base64url key string into CryptoKey object
async function importKey(keyStr) {
    const keyBytes = base64urlDecode(keyStr);
    return crypto.subtle.importKey(
        'raw',
        keyBytes,
        { name: 'AES-GCM', length: 128 },
        false,
        ['encrypt', 'decrypt']
    );
}

// encrypt plaintext string, returns base64url ciphertext
// format: base64url(IV || ciphertext || tag)
// payload: 0x00 || utf8(plaintext)
async function encrypt(plaintext, keyStr) {
    const key = await importKey(keyStr);
    const iv = new Uint8Array(12);
    crypto.getRandomValues(iv);

    // prepend type byte
    const encoder = new TextEncoder();
    const textBytes = encoder.encode(plaintext);
    const payload = new Uint8Array(1 + textBytes.length);
    payload[0] = TYPE_TEXT;
    payload.set(textBytes, 1);

    const ciphertext = await crypto.subtle.encrypt(
        { name: 'AES-GCM', iv: iv },
        key,
        payload
    );

    // combine IV + ciphertext (includes tag)
    const result = new Uint8Array(iv.length + ciphertext.byteLength);
    result.set(iv, 0);
    result.set(new Uint8Array(ciphertext), iv.length);

    return base64urlEncode(result);
}

// decrypt base64url ciphertext, returns plaintext string
// throws on wrong key or corrupted data
async function decrypt(ciphertextStr, keyStr) {
    const key = await importKey(keyStr);
    const data = base64urlDecode(ciphertextStr);

    // extract IV (first 12 bytes) and ciphertext
    const iv = data.slice(0, 12);
    const ciphertext = data.slice(12);

    const payload = await crypto.subtle.decrypt(
        { name: 'AES-GCM', iv: iv },
        key,
        ciphertext
    );

    const payloadBytes = new Uint8Array(payload);

    // check type byte
    if (payloadBytes[0] !== TYPE_TEXT) {
        throw new Error('not a text message');
    }

    // decode text (skip type byte)
    const decoder = new TextDecoder();
    return decoder.decode(payloadBytes.slice(1));
}

// encrypt file with metadata, returns base64url ciphertext
// payload: 0x01 || len_be16(filename) || filename || len_be16(contentType) || contentType || data
// data can be ArrayBuffer or Uint8Array
async function encryptFile(data, filename, contentType, keyStr) {
    const key = await importKey(keyStr);
    const iv = new Uint8Array(12);
    crypto.getRandomValues(iv);

    // normalize to Uint8Array (handles both ArrayBuffer and Uint8Array)
    const fileData = data instanceof Uint8Array ? data : new Uint8Array(data);

    const encoder = new TextEncoder();
    const filenameBytes = encoder.encode(filename);
    const contentTypeBytes = encoder.encode(contentType);

    // calculate payload size: 1 (type) + 2 (filename len) + filename + 2 (ct len) + ct + data
    const payloadSize = 1 + 2 + filenameBytes.length + 2 + contentTypeBytes.length + fileData.length;
    const payload = new Uint8Array(payloadSize);

    let offset = 0;

    // type byte
    payload[offset++] = TYPE_FILE;

    // filename length (big-endian 16-bit)
    payload[offset++] = (filenameBytes.length >> 8) & 0xff;
    payload[offset++] = filenameBytes.length & 0xff;

    // filename
    payload.set(filenameBytes, offset);
    offset += filenameBytes.length;

    // content-type length (big-endian 16-bit)
    payload[offset++] = (contentTypeBytes.length >> 8) & 0xff;
    payload[offset++] = contentTypeBytes.length & 0xff;

    // content-type
    payload.set(contentTypeBytes, offset);
    offset += contentTypeBytes.length;

    // file data
    payload.set(fileData, offset);

    const ciphertext = await crypto.subtle.encrypt(
        { name: 'AES-GCM', iv: iv },
        key,
        payload
    );

    // combine IV + ciphertext
    const result = new Uint8Array(iv.length + ciphertext.byteLength);
    result.set(iv, 0);
    result.set(new Uint8Array(ciphertext), iv.length);

    return base64urlEncode(result);
}

// decrypt file, returns {filename, contentType, data}
// throws on wrong key or corrupted data
async function decryptFile(ciphertextStr, keyStr) {
    const key = await importKey(keyStr);
    const data = base64urlDecode(ciphertextStr);

    // extract IV and ciphertext
    const iv = data.slice(0, 12);
    const ciphertext = data.slice(12);

    const payload = await crypto.subtle.decrypt(
        { name: 'AES-GCM', iv: iv },
        key,
        ciphertext
    );

    const payloadBytes = new Uint8Array(payload);
    let offset = 0;

    // check type byte
    if (payloadBytes[offset++] !== TYPE_FILE) {
        throw new Error('not a file message');
    }

    // read filename length (big-endian)
    const filenameLen = (payloadBytes[offset] << 8) | payloadBytes[offset + 1];
    offset += 2;

    // read filename
    const decoder = new TextDecoder();
    const filename = decoder.decode(payloadBytes.slice(offset, offset + filenameLen));
    offset += filenameLen;

    // read content-type length
    const contentTypeLen = (payloadBytes[offset] << 8) | payloadBytes[offset + 1];
    offset += 2;

    // read content-type
    const contentType = decoder.decode(payloadBytes.slice(offset, offset + contentTypeLen));
    offset += contentTypeLen;

    // rest is file data
    const fileData = payloadBytes.slice(offset);

    return { filename, contentType, data: fileData };
}

// detect content type from decrypted payload (without full decryption)
// useful for UI to show appropriate controls
function getContentType(ciphertextStr, keyStr) {
    // we can't detect without decrypting due to GCM authentication
    // caller should try decrypt() first, then decryptFile() if it fails
    return null;
}

// unified decrypt that auto-detects text vs file
async function decryptAuto(ciphertextStr, keyStr) {
    const key = await importKey(keyStr);
    const data = base64urlDecode(ciphertextStr);

    const iv = data.slice(0, 12);
    const ciphertext = data.slice(12);

    const payload = await crypto.subtle.decrypt(
        { name: 'AES-GCM', iv: iv },
        key,
        ciphertext
    );

    const payloadBytes = new Uint8Array(payload);
    const typeByte = payloadBytes[0];

    if (typeByte === TYPE_TEXT) {
        const decoder = new TextDecoder();
        return { type: 'text', text: decoder.decode(payloadBytes.slice(1)) };
    } else if (typeByte === TYPE_FILE) {
        let offset = 1;

        const filenameLen = (payloadBytes[offset] << 8) | payloadBytes[offset + 1];
        offset += 2;

        const decoder = new TextDecoder();
        const filename = decoder.decode(payloadBytes.slice(offset, offset + filenameLen));
        offset += filenameLen;

        const contentTypeLen = (payloadBytes[offset] << 8) | payloadBytes[offset + 1];
        offset += 2;

        const contentType = decoder.decode(payloadBytes.slice(offset, offset + contentTypeLen));
        offset += contentTypeLen;

        const fileData = payloadBytes.slice(offset);

        return { type: 'file', filename, contentType, data: fileData };
    } else {
        throw new Error('unknown content type: ' + typeByte);
    }
}
