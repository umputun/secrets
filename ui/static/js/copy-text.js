document.querySelectorAll("[data-copy]")
    .forEach(el => {
        let copyBtn = el.querySelector("[data-copy-btn]");

        copyBtn.addEventListener("click", function () {
            const
                textarea = el.querySelector("[data-copy-text]"),
                textToCopy = textarea.value;

            if (textToCopy) {
                const textArea = document.createElement("textarea")
                textArea.value = textToCopy;

                document.body.appendChild(textArea)

                textArea.focus()
                textArea.select()

                document.execCommand('copy')

                document.body.removeChild(textArea)

                let popup = document.getElementById("popup");
                const popupText = popup.querySelector(".popup-text");

                popup.style.opacity = 1;
                popup.style.visibility = "visible";
                popup.style.animation = "fadeInOut 3s ease-in-out";

                if (copyBtn.id === "copy-link-btn") {
                    popupText.innerHTML = `Link copied.<br/>Share this link to access your secret content`
                    return
                }

                popupText.innerHTML = `Message is copied.`

            }
        });
    });