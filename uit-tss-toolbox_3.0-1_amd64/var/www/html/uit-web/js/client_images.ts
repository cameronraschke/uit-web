type ImageManifest = {
	time: Date
	tagnumber: number
	uuid: string
	sha256_hash: string
	filename: string
	filepath: string
	thumbnail_filepath: string
	file_size: number
	mime_type: string
	exif_timestamp: Date
	resolution_x: number
	resolution_y: number
	url: string
	hidden: boolean
	primary_image: boolean
	note: string
	file_type: string
};


const container = document.getElementById('image-container') as HTMLElement;

async function loadClientImages(clientTag: number) {
	if (!container) {
		console.error('Image container not found in DOM.');
		return;
	}

	container.innerHTML = '';
	if (!validateTagInput(clientTag)) {
		const invalidTagParagraph = document.createElement('p');
		invalidTagParagraph.textContent = 'Invalid client tag provided.';
		container.appendChild(invalidTagParagraph);
		return;
	}

	try {
		const response = await fetch(`/api/images/manifest?tagnumber=${encodeURIComponent(clientTag)}`)
		if (!response.ok) {
			if (response.status === 404) {
				const noManifestErrorParagraph = document.createElement('p');
				noManifestErrorParagraph.textContent = `No images found for tag ${clientTag}.`;
				container.appendChild(noManifestErrorParagraph);
				return;
			}
			throw new Error (`Error fetching images: ${response.status} ${response.statusText}`);
		}

		const jsonData = await response.json();
		const imageManifest: ImageManifest[] = jsonData;

		const manifestArr = Array.isArray(imageManifest) ? imageManifest : (imageManifest ? [imageManifest] : []);
    if (manifestArr.length === 0) {
      const noImagesParagraph = document.createElement('p');
      noImagesParagraph.textContent = `No images found for tag ${clientTag}.`;
      container.appendChild(noImagesParagraph);
      return;
    }

    let imageIndex = 1;
    for (const img of manifestArr) {
      const div = document.createElement('div');
      div.className = 'image-entry';
      div.setAttribute('id', `${img.uuid}`);
      if (img.primary_image) {
        div.setAttribute('imageManifest-primary-image', 'true');
        div.style.border = '2px solid black';
        div.style.backgroundColor = 'lightgray';
        const pinnedMessage = document.createElement('p');
        pinnedMessage.textContent = 'Pinned';
        pinnedMessage.style.fontWeight = 'bold';
        div.appendChild(pinnedMessage);
      } else {
        div.setAttribute('imageManifest-primary-image', 'false');
      }
      const primaryImageDiv = document.querySelector(`[imageManifest-primary-image="true"]`);

			const imgDiv = document.createElement('div');
			imgDiv.className = 'image-box';

			const timestampDiv = document.createElement('div');
      timestampDiv.className = 'image-caption';

			const timeStampCaption = document.createElement('p');
			const timeStamp = new Date(img.time);
			if (isNaN(timeStamp.getTime())) {
				timeStampCaption.textContent = 'N/A';
			} else {
				timeStampCaption.textContent = `Uploaded on: ${timeStamp.toLocaleDateString()} ${timeStamp.toLocaleTimeString()}`;
			}

			const imgLink = document.createElement('a');
      const imgURL = new URL(`/api/images`, window.location.origin);
      imgURL.searchParams.set('tagnumber', clientTag.toString());
      imgURL.searchParams.set('uuid', img.uuid);
      imgLink.href = imgURL.toString();
			imgLink.target = '_blank';
			imgLink.rel = 'noopener noreferrer';

      let media = null as HTMLImageElement | HTMLVideoElement | null;
      if (img.mime_type && img.mime_type.startsWith('video/')) {
			  media = document.createElement('video');
			media.controls = true;
      } else if (img.mime_type && img.mime_type.startsWith('image/')) {
        media = document.createElement('img');
      	media.loading = 'lazy';
				media.alt = `Images for ${clientTag}`;
      } else {
        console.warn(`Unsupported media type: ${img.mime_type} for image UUID: ${img.uuid}`);
        continue;
      }
      if (!media) {
        console.warn(`Failed to create media element for image UUID: ${img.uuid}`);
        continue;
      }
      media.src = imgURL.toString();
			media.className = 'client-image';

      const captionDiv = document.createElement('div');
      captionDiv.className = 'image-caption';

			const fileSizeCaption = document.createElement('p');
			if (img.file_size && !isNaN(img.file_size)) {
				const fileSizeInMB = img.file_size / (1024 * 1024);
				if (fileSizeInMB >= 1) {
					fileSizeCaption.textContent = `(size: ${fileSizeInMB.toFixed(2)} MB)`;
				} else {
					const fileSizeInKB = img.file_size / 1024;
					fileSizeCaption.textContent = `(size: ${fileSizeInKB.toFixed(2)} KB)`;
				}
			}

			const noteCaption = document.createElement('p');
			noteCaption.textContent = img.note || "No description";
			if (noteCaption.textContent === "No description") {
				noteCaption.style.fontStyle = "italic";
			}

      const deleteIcon = document.createElement('span');
      deleteIcon.dataset.uuid = img.uuid;
      deleteIcon.dataset.imageCount = imageIndex + "/" + manifestArr.length;
      deleteIcon.className = 'delete-icon';
      deleteIcon.innerHTML = '&times;';
      deleteIcon.title = 'Delete Image';

      const unpinIcon = document.createElement('span');
      unpinIcon.dataset.uuid = img.uuid;
      unpinIcon.className = 'unpin-icon';
      unpinIcon.innerHTML = '&#128204;';
      unpinIcon.style.fontSize = '1rem';
      unpinIcon.title = 'Unpin Image';

      const imageCount = document.createElement('span');
      imageCount.className = 'image-count';
      imageCount.textContent = imageIndex++ + "/" + manifestArr.length || '';

			timestampDiv.appendChild(timeStampCaption);
			imgLink.appendChild(media);
			imgDiv.appendChild(imgLink);
			captionDiv.appendChild(fileSizeCaption);
			captionDiv.appendChild(noteCaption);
      
      captionDiv.appendChild(unpinIcon);
      captionDiv.appendChild(deleteIcon);
      captionDiv.appendChild(imageCount);
			div.appendChild(timestampDiv);
			div.appendChild(imgDiv);
			div.appendChild(captionDiv);
      container.appendChild(div);
      
      if (primaryImageDiv) {
        container.insertBefore(primaryImageDiv, container.firstChild);
      }

			unpinIcon.addEventListener('click', async (event) => {
				const button = event.currentTarget as HTMLInputElement;
				button.disabled = true;
				const uuidToUnpin = button.dataset.uuid;
				if (!uuidToUnpin) {
					alert('Error: No UUID found for this image.');
					return;
				}
				const imageEntry = document.getElementById(uuidToUnpin);

				const currentURL = new URL(window.location.href);
				const clientTag = currentURL.searchParams.get("tagnumber") ? parseInt(currentURL.searchParams.get("tagnumber") as string) : null;				try {
					
					const unpinURL = new URL(`/api/images/toggle_pin`, window.location.origin);
					const unpinResponse = await fetch(unpinURL, {
						method: 'POST',
						headers: { 'Content-Type': 'application/json' },
						credentials: 'same-origin',
						body: JSON.stringify({uuid: uuidToUnpin, tagnumber: clientTag})
					});
					if (!unpinResponse.ok) {
						throw new Error (`Failed to unpin image: ${unpinResponse.status} ${unpinResponse.statusText}`);
					}
					if (imageEntry) {
						imageEntry.style.transition = imageEntry.style.transition || 'opacity 150ms ease';
						imageEntry.style.opacity = '0.5';
						await waitForNextPaint(2);
						imageEntry.removeAttribute('imageManifest-primary-image');
						imageEntry.style.border = 'none';
						imageEntry.style.backgroundColor = 'transparent';
						const pinnedMsg = imageEntry.querySelector('p');
						if (pinnedMsg) {
							pinnedMsg.textContent = "Pinned";
							pinnedMsg.style.fontStyle = "italic";
						}
					}
				} catch (unpinError) {
					alert(`Error unpinning image: ${unpinError.message}`);
					} finally {
					if (imageEntry) imageEntry.style.opacity = '1';
					if (button instanceof HTMLInputElement) button.disabled = false;
				}
			});

      deleteIcon.addEventListener('click', async (event) => {
        const button = event.currentTarget as HTMLInputElement;
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
          deleteURL.searchParams.set('tagnumber', clientTag.toString());
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
		container.innerHTML = '';
		const errorParagraph = document.createElement('p');
		errorParagraph.textContent = `Error fetching images: ${err.message}`;
		container.appendChild(errorParagraph);
    console.warn(`Error fetching images for tag ${clientTag}: ${err.message}`);
	}
}

document.addEventListener('DOMContentLoaded', async () => {
	const urlParams = new URLSearchParams(window.location.search);
	const clientTag = urlParams.get('tagnumber');
	  if (!clientTag) {
	    console.warn('No tagnumber parameter found in URL.');
	    return;
	  }
		const clientTagNumber = parseInt(clientTag, 10); // more strict validation happens in loadClientImages
		await loadClientImages(clientTagNumber);
});