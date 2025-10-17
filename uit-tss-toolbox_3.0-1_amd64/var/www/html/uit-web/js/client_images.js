async function loadClientImages(clientTag) {
	try {
		const container = document.getElementById('image-container');
		container.innerHTML = '';
		if (!clientTag) {
			container.innerHTML = '<p>Please provide a valid client tag.</p>';
			return;
		}
		const response = await fetch(`/api/images/manifest?tagnumber=${encodeURIComponent(clientTag)}`)
		if (!response.ok) {
			if (response.status === 404) {
				container.innerHTML = '<p>No images found for this client tag.</p>';
				return;
			}
			throw new Error(`HTTP error! status: ${response.status}`);
		}
		const data = await response.json();
		if (data.error) {
			container.innerHTML = `<p>Error: ${data.error}</p>`;
			return;
		}
		if (data.length === 0) {
			container.innerHTML = '<p>No images found for this client tag.</p>';
			return;
		}
		data.forEach(imgJsonManifest => {
			const imgDiv = document.createElement('div');
			imgDiv.className = 'image-box';

			const imgLink = document.createElement('a');
			imgLink.href = `/api/images/${imgJsonManifest.UUID}`;
			imgLink.target = '_blank';
			imgLink.rel = 'noopener noreferrer';

			const img = document.createElement('img');
			img.src = `/api/images/${imgJsonManifest.UUID}`;
			img.alt = `Image for ${clientTag}`;
			img.className = 'client-image';

			const timeStampCaption = document.createElement('p');
			const timeStamp = new Date(imgJsonManifest.Time);
			timeStampCaption.textContent = `Uploaded on: ${timeStamp.toLocaleDateString()} ${timeStamp.toLocaleTimeString()}`;

			const noteCaption = document.createElement('p');
			noteCaption.textContent = imgJsonManifest.Note || "No description";
			if (noteCaption.textContent === "No description") {
				noteCaption.style.fontStyle = "italic";
			}

			imgLink.appendChild(img);
			imgDiv.appendChild(imgLink);
			imgDiv.appendChild(timeStampCaption);
			imgDiv.appendChild(noteCaption);
			container.appendChild(imgDiv);
		});
	} catch (err) {
		container.innerHTML = `<p>Error fetching images: ${err.message}</p>`;
	}
}

document.addEventListener('DOMContentLoaded', async () => {
    const urlParams = new URLSearchParams(window.location.search);
    const clientTag = urlParams.get('tagnumber');
    if (clientTag && clientTag.length === 6) {
      await loadClientImages(clientTag);
    } else {
      console.warn('No valid tagnumber parameter found in URL.');
    }
});