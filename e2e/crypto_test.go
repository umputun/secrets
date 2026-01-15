//go:build e2e

package e2e

import (
	"testing"

	"github.com/playwright-community/playwright-go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// waitForCryptoJS waits for crypto.js to be loaded (it's included in the page automatically)
func waitForCryptoJS(t *testing.T, page playwright.Page) {
	t.Helper()
	// crypto.js is loaded via <script src> in index.tmpl.html, wait for it to be available
	_, err := page.WaitForFunction("typeof generateKey === 'function'", nil)
	require.NoError(t, err, "crypto.js should be loaded")
}

func TestCrypto_CheckAvailable(t *testing.T) {
	page := newPage(t)
	_, err := page.Goto(baseURL)
	require.NoError(t, err)

	waitForCryptoJS(t, page)

	// test checkCryptoAvailable
	result, err := page.Evaluate("checkCryptoAvailable()")
	require.NoError(t, err)
	assert.True(t, result.(bool), "Web Crypto should be available on localhost")
}

func TestCrypto_GenerateKey(t *testing.T) {
	page := newPage(t)
	_, err := page.Goto(baseURL)
	require.NoError(t, err)

	waitForCryptoJS(t, page)

	// test generateKey
	result, err := page.Evaluate("(async () => { return await generateKey(); })()")
	require.NoError(t, err)

	key := result.(string)
	assert.Len(t, key, 22, "key should be 22 chars (128-bit base64url)")
	assert.Regexp(t, `^[A-Za-z0-9_-]+$`, key, "key should be base64url encoded")
}

func TestCrypto_TextRoundTrip(t *testing.T) {
	page := newPage(t)
	_, err := page.Goto(baseURL)
	require.NoError(t, err)

	waitForCryptoJS(t, page)

	// test encrypt/decrypt round-trip
	result, err := page.Evaluate(`(async () => {
		const key = await generateKey();
		const plaintext = 'hello world, this is a secret message!';
		const ciphertext = await encrypt(plaintext, key);
		const decrypted = await decrypt(ciphertext, key);
		return { plaintext, ciphertext, decrypted, key };
	})()`)
	require.NoError(t, err)

	data := result.(map[string]interface{})
	assert.Equal(t, data["plaintext"], data["decrypted"], "decrypted should match original")
	assert.NotEqual(t, data["plaintext"], data["ciphertext"], "ciphertext should differ from plaintext")
}

func TestCrypto_TextWrongKey(t *testing.T) {
	page := newPage(t)
	_, err := page.Goto(baseURL)
	require.NoError(t, err)

	waitForCryptoJS(t, page)

	// test decrypt with wrong key should fail
	result, err := page.Evaluate(`(async () => {
		const key1 = await generateKey();
		const key2 = await generateKey();
		const ciphertext = await encrypt('secret message', key1);
		try {
			await decrypt(ciphertext, key2);
			return { error: null };
		} catch (e) {
			return { error: e.message || e.toString() };
		}
	})()`)
	require.NoError(t, err)

	data := result.(map[string]interface{})
	assert.NotNil(t, data["error"], "decryption with wrong key should fail")
}

func TestCrypto_FileRoundTrip(t *testing.T) {
	page := newPage(t)
	_, err := page.Goto(baseURL)
	require.NoError(t, err)

	waitForCryptoJS(t, page)

	// test file encrypt/decrypt round-trip
	result, err := page.Evaluate(`(async () => {
		const key = await generateKey();
		const filename = 'test-document.pdf';
		const contentType = 'application/pdf';
		const fileData = new Uint8Array([0x25, 0x50, 0x44, 0x46, 0x2d, 0x31, 0x2e, 0x34]); // %PDF-1.4

		const ciphertext = await encryptFile(fileData, filename, contentType, key);
		const decrypted = await decryptFile(ciphertext, key);

		return {
			originalFilename: filename,
			originalContentType: contentType,
			originalDataLength: fileData.length,
			decryptedFilename: decrypted.filename,
			decryptedContentType: decrypted.contentType,
			decryptedDataLength: decrypted.data.length,
			dataMatches: JSON.stringify(Array.from(fileData)) === JSON.stringify(Array.from(decrypted.data))
		};
	})()`)
	require.NoError(t, err)

	data := result.(map[string]interface{})
	assert.Equal(t, "test-document.pdf", data["decryptedFilename"])
	assert.Equal(t, "application/pdf", data["decryptedContentType"])
	assert.Equal(t, data["originalDataLength"], data["decryptedDataLength"])
	assert.True(t, data["dataMatches"].(bool), "file data should match after round-trip")
}

func TestCrypto_FileRoundTrip_ArrayBuffer(t *testing.T) {
	page := newPage(t)
	_, err := page.Goto(baseURL)
	require.NoError(t, err)

	waitForCryptoJS(t, page)

	// test file encrypt/decrypt with ArrayBuffer input (from File.arrayBuffer())
	result, err := page.Evaluate(`(async () => {
		const key = await generateKey();
		const filename = 'arraybuffer-test.bin';
		const contentType = 'application/octet-stream';
		// simulate File.arrayBuffer() by creating ArrayBuffer directly
		const arrayBuffer = new Uint8Array([1, 2, 3, 4, 5]).buffer;

		const ciphertext = await encryptFile(arrayBuffer, filename, contentType, key);
		const decrypted = await decryptFile(ciphertext, key);

		return {
			originalDataLength: arrayBuffer.byteLength,
			decryptedFilename: decrypted.filename,
			decryptedDataLength: decrypted.data.length,
			dataMatches: decrypted.data[0] === 1 && decrypted.data[4] === 5
		};
	})()`)
	require.NoError(t, err)

	data := result.(map[string]interface{})
	assert.Equal(t, "arraybuffer-test.bin", data["decryptedFilename"])
	assert.EqualValues(t, 5, data["originalDataLength"])
	assert.EqualValues(t, 5, data["decryptedDataLength"])
	assert.True(t, data["dataMatches"].(bool), "ArrayBuffer data should round-trip correctly")
}

func TestCrypto_DecryptAuto_Text(t *testing.T) {
	page := newPage(t)
	_, err := page.Goto(baseURL)
	require.NoError(t, err)

	waitForCryptoJS(t, page)

	// test decryptAuto detects text
	result, err := page.Evaluate(`(async () => {
		const key = await generateKey();
		const ciphertext = await encrypt('hello world', key);
		const result = await decryptAuto(ciphertext, key);
		return result;
	})()`)
	require.NoError(t, err)

	data := result.(map[string]interface{})
	assert.Equal(t, "text", data["type"])
	assert.Equal(t, "hello world", data["text"])
}

func TestCrypto_DecryptAuto_File(t *testing.T) {
	page := newPage(t)
	_, err := page.Goto(baseURL)
	require.NoError(t, err)

	waitForCryptoJS(t, page)

	// test decryptAuto detects file
	result, err := page.Evaluate(`(async () => {
		const key = await generateKey();
		const fileData = new Uint8Array([1, 2, 3, 4, 5]);
		const ciphertext = await encryptFile(fileData, 'test.bin', 'application/octet-stream', key);
		const result = await decryptAuto(ciphertext, key);
		return {
			type: result.type,
			filename: result.filename,
			contentType: result.contentType,
			dataLength: result.data.length
		};
	})()`)
	require.NoError(t, err)

	data := result.(map[string]interface{})
	assert.Equal(t, "file", data["type"])
	assert.Equal(t, "test.bin", data["filename"])
	assert.Equal(t, "application/octet-stream", data["contentType"])
	assert.EqualValues(t, 5, data["dataLength"])
}

func TestCrypto_UnicodePlaintext(t *testing.T) {
	page := newPage(t)
	_, err := page.Goto(baseURL)
	require.NoError(t, err)

	waitForCryptoJS(t, page)

	// test unicode text round-trip
	result, err := page.Evaluate(`(async () => {
		const key = await generateKey();
		const plaintext = '你好世界! 🔐 Привет мир! مرحبا بالعالم';
		const ciphertext = await encrypt(plaintext, key);
		const decrypted = await decrypt(ciphertext, key);
		return { plaintext, decrypted };
	})()`)
	require.NoError(t, err)

	data := result.(map[string]interface{})
	assert.Equal(t, data["plaintext"], data["decrypted"], "unicode text should round-trip correctly")
}

func TestCrypto_UnicodeFilename(t *testing.T) {
	page := newPage(t)
	_, err := page.Goto(baseURL)
	require.NoError(t, err)

	waitForCryptoJS(t, page)

	// test unicode filename round-trip
	result, err := page.Evaluate(`(async () => {
		const key = await generateKey();
		const filename = '文档_документ_📄.pdf';
		const contentType = 'application/pdf';
		const fileData = new Uint8Array([1, 2, 3]);

		const ciphertext = await encryptFile(fileData, filename, contentType, key);
		const decrypted = await decryptFile(ciphertext, key);

		return {
			originalFilename: filename,
			decryptedFilename: decrypted.filename
		};
	})()`)
	require.NoError(t, err)

	data := result.(map[string]interface{})
	assert.Equal(t, data["originalFilename"], data["decryptedFilename"], "unicode filename should round-trip correctly")
}

func TestCrypto_EmptyPlaintext(t *testing.T) {
	page := newPage(t)
	_, err := page.Goto(baseURL)
	require.NoError(t, err)

	waitForCryptoJS(t, page)

	// test empty string
	result, err := page.Evaluate(`(async () => {
		const key = await generateKey();
		const plaintext = '';
		const ciphertext = await encrypt(plaintext, key);
		const decrypted = await decrypt(ciphertext, key);
		return { plaintext, decrypted, ciphertextLength: ciphertext.length };
	})()`)
	require.NoError(t, err)

	data := result.(map[string]interface{})
	assert.Equal(t, "", data["decrypted"], "empty string should decrypt to empty string")
	// use type switch since JS may return int or float64
	var ciphertextLen float64
	switch v := data["ciphertextLength"].(type) {
	case float64:
		ciphertextLen = v
	case int:
		ciphertextLen = float64(v)
	}
	assert.Greater(t, ciphertextLen, float64(0), "ciphertext should not be empty")
}

func TestCrypto_LargeBinaryFile(t *testing.T) {
	page := newPage(t)
	_, err := page.Goto(baseURL)
	require.NoError(t, err)

	waitForCryptoJS(t, page)

	// test large binary file (100KB)
	result, err := page.Evaluate(`(async () => {
		const key = await generateKey();
		const size = 100 * 1024; // 100KB
		const fileData = new Uint8Array(size);
		for (let i = 0; i < size; i++) {
			fileData[i] = i % 256;
		}

		const ciphertext = await encryptFile(fileData, 'large.bin', 'application/octet-stream', key);
		const decrypted = await decryptFile(ciphertext, key);

		// verify data integrity
		let matches = true;
		for (let i = 0; i < size; i++) {
			if (decrypted.data[i] !== fileData[i]) {
				matches = false;
				break;
			}
		}

		return {
			originalSize: size,
			decryptedSize: decrypted.data.length,
			dataMatches: matches
		};
	})()`)
	require.NoError(t, err)

	data := result.(map[string]interface{})
	assert.EqualValues(t, 100*1024, data["originalSize"])
	assert.EqualValues(t, data["originalSize"], data["decryptedSize"])
	assert.True(t, data["dataMatches"].(bool), "large file data should match after round-trip")
}
