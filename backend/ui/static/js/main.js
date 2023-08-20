document.getElementById("close-popup").addEventListener("click", function () {
    let popup = document.getElementById("popup");
    popup.style.transition = "visibility 0s linear 0.33s, opacity 0.33s linear;";
    popup.style.opacity = 0;
    popup.style.visibility = "hidden";
});