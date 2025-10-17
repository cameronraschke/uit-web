function loadClientImages(clientTag) {
    const container = document.getElementById('image-container');
    container.innerHTML = '';
    if (!clientTag) {
        container.innerHTML = '<p>Please provide a valid client tag.</p>';
        return;
    }
    fetch(`/api/images/manifest?tagnumber=${encodeURIComponent(clientTag)}`)
        .then(response => response.json())
        .then(data => {
            if (data.error) {
                container.innerHTML = `<p>Error: ${data.error}</p>`;
                return;
            }
            if (data.length === 0) {
                container.innerHTML = '<p>No images found for this client tag.</p>';
                return;
            }
            console.log(typeof data);
            console.log(data);
            data.forEach(imgUrl => {
                console.log(imgUrl.UUID);
                const img = document.createElement('img');
                img.src = `/api/images/${imgUrl.UUID}`;
                img.alt = `Image for ${clientTag}`;
                img.className = 'client-image';
                container.appendChild(img);
            });
        })
        .catch(err => {
            container.innerHTML = `<p>Error fetching images: ${err.message}</p>`;
        });
}

document.addEventListener('DOMContentLoaded', () => {
    const urlParams = new URLSearchParams(window.location.search);
    const clientTag = urlParams.get('tagnumber');
    loadClientImages(clientTag);
});