async function loadClientImages(clientTag) {
  const container = document.getElementById('image-container');
	try {
		container.innerHTML = '';
		if (!clientTag) {
			container.innerHTML = '<p>Please provide a valid client tag.</p>';
			return;
		}

		const response = await fetch(`/api/images/manifest?tagnumber=${encodeURIComponent(clientTag)}`)
		if (!response.ok) {
			if (response.status === 404) {
				container.innerHTML = `<p>No images found for tag ${clientTag}.</p>`;
				return;
			}
			throw new Error (`Error fetching images: ${response.status} ${response.statusText}`);
		}

    let data = null;
    try {
      const contentType = response.headers.get('Content-Type') || '';
      if (contentType.includes('application/json')) {
        data = await response.json();
      } else {
        const textData = await response.text();
        data = textData.trim() ? JSON.parse(textData) : [];
      }
    } catch (parseError) {
      throw new Error(`Failed to parse response JSON: ${parseError.message}`);
    }

		if (data && typeof data === 'object' && !Array.isArray(data) && Object.prototype.hasOwnProperty.call(data, 'error')) {
      throw new Error(String(data.error || 'Unknown server error'));
    }
		const items = Array.isArray(data) ? data : (data ? [data] : []);
    if (items.length === 0) {
      container.innerHTML = `<p>No images found for tag ${clientTag}.</p>`;
      return;
    }
    let imageIndex = 1;
    for (const imgJsonManifest of items) {
      const div = document.createElement('div');
      div.className = 'image-entry';
      div.setAttribute('id', `${imgJsonManifest.uuid}`);
      if (imgJsonManifest.primary_image) {
        div.setAttribute('data-primary-image', 'true');
        div.style.border = '2px solid black';
        div.style.backgroundColor = 'lightgray';
        const pinnedMessage = document.createElement('p');
        pinnedMessage.textContent = 'Pinned';
        pinnedMessage.style.fontWeight = 'bold';
        div.appendChild(pinnedMessage);
      } else {
        div.setAttribute('data-primary-image', 'false');
      }
      const primaryImageDiv = document.querySelector(`[data-primary-image="true"]`);

			const imgDiv = document.createElement('div');
			imgDiv.className = 'image-box';

			const imgLink = document.createElement('a');
      const imgURL = new URL(`/api/images`, window.location.origin);
      imgURL.searchParams.set('tagnumber', clientTag);
      imgURL.searchParams.set('uuid', imgJsonManifest.uuid);
      imgLink.href = imgURL.toString();
			imgLink.target = '_blank';
			imgLink.rel = 'noopener noreferrer';

      let media = null;
      if (imgJsonManifest.file_type && imgJsonManifest.file_type.startsWith('video/')) {
			  media = document.createElement('video');
			media.controls = true;
      } else if (imgJsonManifest.file_type && imgJsonManifest.file_type.startsWith('image/')) {
        media = document.createElement('img');
      } else {
        console.warn(`Unsupported media type: ${imgJsonManifest.file_type} for image UUID: ${imgJsonManifest.uuid}`);
        continue;
      }
      if (!media) {
        console.warn(`Failed to create media element for image UUID: ${imgJsonManifest.uuid}`);
        continue;
      }
      media.lazy = true;
      media.src = imgURL.toString();
			media.alt = `Media for ${clientTag}`;
			media.className = 'client-image';

      const captionDiv = document.createElement('div');
      captionDiv.className = 'image-caption';

			const timeStampCaption = document.createElement('p');
			const timeStamp = new Date(imgJsonManifest.time);
      if (isNaN(timeStamp.getTime())) {
        timeStampCaption.textContent = 'Uploaded on: Unknown date';
      } else {
        timeStampCaption.textContent = `Uploaded on: ${timeStamp.toLocaleDateString()} ${timeStamp.toLocaleTimeString()}`;
      }

			const noteCaption = document.createElement('p');
			noteCaption.textContent = imgJsonManifest.note || "No description";
			if (noteCaption.textContent === "No description") {
				noteCaption.style.fontStyle = "italic";
			}


      const deleteIcon = document.createElement('span');
      deleteIcon.dataset.uuid = imgJsonManifest.uuid;
      deleteIcon.dataset.imageCount = imageIndex + "/" + items.length;
      deleteIcon.className = 'delete-icon';
      deleteIcon.innerHTML = '&times;';
      deleteIcon.title = 'Delete Image';

      const unpinIcon = document.createElement('span');
      unpinIcon.dataset.uuid = imgJsonManifest.uuid;
      unpinIcon.className = 'unpin-icon';
      unpinIcon.innerHTML = '&#128204;';
      unpinIcon.style.fontSize = '1rem';
      unpinIcon.title = 'Unpin Image';

      const imageCount = document.createElement('span');
      imageCount.className = 'image-count';
      imageCount.textContent = imageIndex++ + "/" + items.length || '';

			imgLink.appendChild(media);
			imgDiv.appendChild(imgLink);
			captionDiv.appendChild(timeStampCaption);
			captionDiv.appendChild(noteCaption);
      
      captionDiv.appendChild(unpinIcon);
      captionDiv.appendChild(deleteIcon);
      captionDiv.appendChild(imageCount);
			div.appendChild(imgDiv);
			div.appendChild(captionDiv);
      container.appendChild(div);
      
      if (primaryImageDiv) {
        container.insertBefore(primaryImageDiv, container.firstChild);
      }

      unpinIcon.addEventListener('click', async (event) => {
        const button = event.currentTarget;
        if (!(button instanceof HTMLElement)) return;
        button.disabled = true;
        const uuidToUnpin = button.dataset.uuid;
        if (!uuidToUnpin) {
          alert('Error: No UUID found for this image.');
          return;
        }
        const entry = document.getElementById(uuidToUnpin);
        try {
          const unpinURL = new URL(`/api/images/toggle_pin`, window.location.origin);
          const unpinResponse = await fetch(unpinURL, {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            credentials: 'same-origin',
            body: JSON.stringify({uuid: uuidToUnpin, tagnumber: Number(clientTag)})
          });
          if (!unpinResponse.ok) {
            throw new Error (`Failed to unpin image: ${unpinResponse.status} ${unpinResponse.statusText}`);
          }
          if (entry) {
            entry.style.transition = entry.style.transition || 'opacity 150ms ease';
            entry.style.opacity = '0.5';
            await waitForNextPaint(2);
            entry.removeAttribute('data-primary-image');
            entry.style.border = 'none';
            entry.style.backgroundColor = 'transparent';
            const pinnedMsg = entry.querySelector('p');
            if (pinnedMsg) {
              pinnedMsg.textContent = "Pinned";
              pinnedMsg.style.fontStyle = "italic";
            }
          }
        } catch (unpinError) {
          alert(`Error unpinning image: ${unpinError.message}`);
        } finally {
          if (entry) entry.style.opacity = '1';
        }
      });

      deleteIcon.addEventListener('click', async (event) => {
        const button = event.currentTarget;
        if (!(button instanceof HTMLElement)) return;
        button.disabled = true;
        const uuidToDelete = button.dataset.uuid;
        if (!uuidToDelete) {
          alert('Error: No UUID found for this image.');
          button.disabled = false;
          return;
        }
        const imageCount = button.dataset.imageCount || '';

        const entry = document.getElementById(uuidToDelete);
        if (entry) {
          entry.style.transition = entry.style.transition || 'opacity 150ms ease';
          entry.style.opacity = '0.5';
          await waitForNextPaint(2);
        }

        const okToDelete = window.confirm(`Are you sure you want to delete this image (${imageCount})?`);
        if (!okToDelete) {
          if (entry) entry.style.opacity = '1';
          button.disabled = false;
          return;
        }

        try {
          const deleteURL = new URL(`/api/images`, window.location.origin);
          deleteURL.searchParams.set('tagnumber', clientTag);
          deleteURL.searchParams.set('uuid', uuidToDelete);
          const deleteResponse = await fetch(deleteURL, {
            method: 'DELETE',
            credentials: 'same-origin'
          });
          if (!deleteResponse.ok) {
            throw new Error (`Failed to delete image: ${deleteResponse.status} ${deleteResponse.statusText}`);
          }
          if (entry) entry.remove();
        } catch (deleteError) {
          alert(`Error deleting image: ${deleteError.message}`);
        } finally {
          if (entry) entry.style.opacity = '1';
          button.disabled = false;
        }
      });
    } 
	} catch (err) {
    if (err.name === 'AbortError') {
      console.warn('Image fetch aborted');
      return;
    }
		container.innerHTML = `<p>Error fetching images: ${err.message}</p>`;
    console.warn(`Error fetching images for tag ${clientTag}: ${err.message}`);
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