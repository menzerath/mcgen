// define all elements
const title = document.querySelector('input[name="title"]');
const text = document.querySelector('input[name="text"]');
const background = document.querySelector('select[name="background"]');
const achievement = document.getElementById('achievement');

const boxURL = document.getElementById('out-url');
const boxHTML = document.getElementById('out-html');
const boxBB = document.getElementById('out-bb');

// handle input change by updating the image
title.oninput = updateImageAfterTimeout;
text.oninput = updateImageAfterTimeout;
background.onchange = updateImage;

// automatically select input field content on click
title.onclick = title.select;
text.onclick = text.select;

// copy link box content to clipboard on click
boxURL.onclick = function () {
    copyURL(boxURL);
}
boxHTML.onclick = function () {
    copyURL(boxHTML);
}
boxBB.onclick = function () {
    copyURL(boxBB);
}

// download image on click
achievement.onclick = function () {
    window.location.href = achievement.src + '&output=download';
}

// call updateImage after inputs stopped for 300ms
let inputTimeout = null;
function updateImageAfterTimeout() {
    clearTimeout(inputTimeout);
    inputTimeout = setTimeout(updateImage, 300);
}

// update image and boxes
function updateImage() {
    achievement.src = `api/v1/achievement?background=${background.value}&title=${encodeURIComponent(title.value)}&text=${encodeURIComponent(text.value)}`;

    boxURL.value = achievement.src;
    boxHTML.value = `<a href="${window.location.href}" target="_blank"><img src="${achievement.src}" alt="Minecraft Achievement" /></a>`;
    boxBB.value = `[url=${window.location.href}][img]${achievement.src}[/img][/url]`;
}

// copy input field content to clipboard
function copyURL(input) {
    input.select();
    navigator.clipboard.writeText(input.value);
}
