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
    let imageIndex = 1;
		data.forEach(imgJsonManifest => {
      const div = document.createElement('div');
      div.className = 'image-entry';
      div.setAttribute('id', `${imgJsonManifest.UUID}`);

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

      const captionDiv = document.createElement('div');
      captionDiv.className = 'image-caption';
			const timeStampCaption = document.createElement('p');
			const timeStamp = new Date(imgJsonManifest.Time);
			timeStampCaption.textContent = `Uploaded on: ${timeStamp.toLocaleDateString()} ${timeStamp.toLocaleTimeString()}`;

			const noteCaption = document.createElement('p');
			noteCaption.textContent = imgJsonManifest.Note || "No description";
			if (noteCaption.textContent === "No description") {
				noteCaption.style.fontStyle = "italic";
			}

      const deleteIcon = document.createElement('span');
      deleteIcon.dataset.uuid = imgJsonManifest.UUID;
      deleteIcon.dataset.imageCount = imageIndex + "/" + data.length;
      deleteIcon.className = 'delete-icon';
      deleteIcon.innerHTML = '&times;';
      deleteIcon.title = 'Delete Image';

      const imageCount = document.createElement('span');
      imageCount.className = 'image-count';
      imageCount.textContent = imageIndex++ + "/" + data.length || '';

			imgLink.appendChild(img);
			imgDiv.appendChild(imgLink);
			captionDiv.appendChild(timeStampCaption);
			captionDiv.appendChild(noteCaption);
      captionDiv.appendChild(deleteIcon);
      captionDiv.appendChild(imageCount);
			div.appendChild(imgDiv);
			div.appendChild(captionDiv);
			container.appendChild(div);

      deleteIcon.addEventListener('click', async (event) => {
        const uuidToDelete = event.target.dataset.uuid;
        const div = document.getElementById(uuidToDelete);

        if (div) {
          div.style.transition = div.style.transition || 'opacity 150ms ease';
          div.style.opacity = '0.5';
          await waitForNextPaint(2);
        }

        const ok = window.confirm(`Are you sure you want to delete this image (${event.target.dataset.imageCount})?`);
        if (!ok) {
          if (div) div.style.opacity = '1';
          return;
        }

        try {
          const deleteResponse = await fetch(`/api/images/${uuidToDelete}`, {
            method: 'DELETE',
            credentials: 'same-origin'
          });
          if (!deleteResponse.ok) {
            console.error(`Failed to delete image: ${deleteResponse.statusText}`);
            if (div) div.style.opacity = '1';
            return;
          }
          if (div) div.remove();
        } catch (error) {
          alert(`Error deleting image: ${error.message}`);
        } finally {
          if (div) div.style.opacity = '1';
        }
      });
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