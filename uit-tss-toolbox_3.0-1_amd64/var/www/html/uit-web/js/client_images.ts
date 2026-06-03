type ImageManifest = {
	time: Date
	client_uuid: string
	file_uuid: string
	sha256_hash: string
	file_size: number
	mime_type: string
	resolution_x: number
	resolution_y: number
	url: string
	hidden: boolean
	pinned: boolean
	caption: string
};


const container = document.getElementById('image-container') as HTMLElement;

async function fetchManifestData(clientUUID: string) : Promise<ImageManifest[]> {
	if (!container) {
		console.error('Image container not found in DOM.');
		return [];
	}

	container.innerHTML = '';
	if (!clientUUID) {
		const invalidTagParagraph = document.createElement('p');
		invalidTagParagraph.textContent = 'Invalid client UUID provided.';
		container.appendChild(invalidTagParagraph);
		return [];
	}
	const manifestURL = new URL(`/api/client/files/manifest`, window.location.origin);
	manifestURL.searchParams.set('tagnumber', clientUUID.toString());

	try {
		const response = await fetch(manifestURL.toString());
		if (!response.ok) {
			if (response.status === 404) {
				return [];
			}
			throw new Error (`Error fetching images: ${response.status} ${response.statusText}`);
		}

		const jsonData = await response.json();
		const imageManifest: ImageManifest[] = jsonData;

		const manifestArr = Array.isArray(imageManifest) ? imageManifest : (imageManifest ? [imageManifest] : []);
    if (manifestArr.length === 0) {
      return [];
    }

		manifestArr.sort((a, b) => {
			const timeA = new Date(a.time).getTime();
			const timeB = new Date(b.time).getTime();
			return timeB - timeA;
		});

		manifestArr.sort((a, b) => {
			if (a.pinned && !b.pinned) return -1;
			if (!a.pinned && b.pinned) return 1;
			return 0;
		});
		return manifestArr;
	} catch (err) {
		if (err instanceof Error) {
			container.innerHTML = '';
			const errorParagraph = document.createElement('p');
			errorParagraph.textContent = `Error fetching images: ${err.message}`;
			container.appendChild(errorParagraph);
		}
		return [];
	}
}

function renderFiles(manifestArr: ImageManifest[], clientUUID: string) {
	let imageIndex = 1;
	for (const file of manifestArr) {
		const fileEntry = document.createElement('div');
		fileEntry.className = 'file-entry';
		fileEntry.setAttribute('id', `${file.file_uuid}`);
		if (file.pinned) {
			fileEntry.classList.add('file-entry', 'primary');
			const pinnedMessage = document.createElement('p');
			pinnedMessage.textContent = 'Pinned';
			fileEntry.appendChild(pinnedMessage);
		} else {
			fileEntry.classList.add('file-entry');
		}

		const iconContainer = document.createElement('div');
		iconContainer.classList.add('file-icons');
		const imageCount = document.createElement('span');
		imageCount.classList.add('smaller-text');
		imageCount.textContent = imageIndex++ + "/" + manifestArr.length || '';
		iconContainer.appendChild(imageCount);

		const unpinIcon = document.createElement('button');
		unpinIcon.dataset.uuid = file.file_uuid;
		if (file.pinned) {
			unpinIcon.classList.add('svg-button', 'pinned');
			unpinIcon.textContent = 'Unpin Image';
		} else {
			unpinIcon.classList.add('svg-button', 'unpinned');
			unpinIcon.textContent = 'Pin Image';
		}
		iconContainer.appendChild(unpinIcon);

		const deleteIcon = document.createElement('button');
		deleteIcon.dataset.uuid = file.file_uuid;
		deleteIcon.dataset.imageCount = imageIndex + "/" + manifestArr.length;
		deleteIcon.classList.add('svg-button', 'delete');
		deleteIcon.title = 'Delete Image';
		iconContainer.appendChild(deleteIcon);
		
		initListeners(unpinIcon, deleteIcon, clientUUID);

		const timestampContainer = document.createElement('div');
		timestampContainer.classList.add('file-caption', 'timestamp');

		const timeStampCaption = document.createElement('p');
		const timeStamp = new Date(file.time);
		if (!isNaN(timeStamp.getTime())) {
			timeStampCaption.textContent = `Uploaded on: ${timeStamp.toLocaleDateString()} ${timeStamp.toLocaleTimeString()}`;
			timeStampCaption.style.fontStyle = "normal";
		} else {
			timeStampCaption.textContent = `Uploaded on: Unknown date`;
			timeStampCaption.style.fontStyle = "italic";
		}
		timestampContainer.appendChild(timeStampCaption);

		const filePreviewContainer = document.createElement('div');
		filePreviewContainer.className = 'file-preview';

		// Source URL
		const imgURL = new URL(`/api/client/files`, window.location.origin);
		imgURL.searchParams.set('client_uuid', file.client_uuid);
		imgURL.searchParams.set('file_uuid', file.file_uuid);

		let filePreview = null as HTMLImageElement | HTMLVideoElement | null;
		if (file.mime_type && file.mime_type.startsWith('video/')) {
			filePreview = document.createElement('video');
			filePreview.controls = true;
			filePreview.preload = 'metadata';
		filePreviewContainer.appendChild(filePreview);
		} else if (file.mime_type && file.mime_type.startsWith('image/')) {
			// Videos do not get an imgLink
			filePreview = document.createElement('img');
			filePreview.loading = 'lazy';
			filePreview.alt = `Images for ${clientUUID}`;
			const imgLink = document.createElement('a');
			imgLink.href = imgURL.toString();
			imgLink.target = '_blank';
			imgLink.rel = 'noopener noreferrer';
			imgLink.appendChild(filePreview);
			filePreviewContainer.appendChild(imgLink);
		} else {
			console.warn(`Unsupported media type: ${file.mime_type} for image UUID: ${file.file_uuid}`);
			continue;
		}
		if (!filePreview) {
			console.warn(`Failed to create media element for image UUID: ${file.file_uuid}`);
			continue;
		}
		filePreview.src = imgURL.toString();
		filePreview.className = 'file-preview';

		const captionContainer = document.createElement('div');
		captionContainer.className = 'file-caption';

		const fileSizeCaption = document.createElement('p');
		fileSizeCaption.classList.add('file-caption', 'size');
		if (file.file_size && !isNaN(file.file_size)) {
			const fileSizeInMB = file.file_size / (1024 * 1024);
			if (fileSizeInMB >= 1) {
				fileSizeCaption.textContent = `(size: ${fileSizeInMB.toFixed(2)} MB)`;
			} else {
				const fileSizeInKB = file.file_size / 1024;
				fileSizeCaption.textContent = `(size: ${fileSizeInKB.toFixed(2)} KB)`;
			}
		} else {
			fileSizeCaption.textContent = '(size: Unknown)';
			fileSizeCaption.style.fontStyle = "italic";
		}

		const noteCaption = document.createElement('p');
		if (file.caption && file.caption.trim() !== '') {
			noteCaption.textContent = file.caption;
			noteCaption.style.fontStyle = "normal";
		} else {
			noteCaption.textContent = "No description";
			noteCaption.style.fontStyle = "italic";
		}
		
		captionContainer.appendChild(noteCaption);
		captionContainer.appendChild(fileSizeCaption);

		fileEntry.appendChild(iconContainer);
		fileEntry.appendChild(timestampContainer);
		fileEntry.appendChild(filePreviewContainer);
		fileEntry.appendChild(captionContainer);


		container.appendChild(fileEntry);
	}
}

function initListeners(unpinEl: HTMLButtonElement, deleteEl: HTMLButtonElement, clientUUID: string) {
	unpinEl.addEventListener('click', async (event) => {
		if (!(unpinEl instanceof HTMLElement)) return;
		const el = event.currentTarget as HTMLButtonElement;
		el.disabled = true;
		const uuidToUnpin = el.dataset.uuid || "";
		if (!uuidToUnpin) {
			alert('Error: No UUID found for this image.');
			el.disabled = false;
			return;
		}
		const currentURL = new URL(window.location.href);
		const clientUUID = currentURL.searchParams.get("client_uuid") ?? null;
		const unpinURL = new URL(`/api/files/toggle_pin`, window.location.origin);
		try {
			const unpinRequest = await fetch(unpinURL, {
				method: 'POST',
				headers: { 'Content-Type': 'application/json' },
				credentials: 'same-origin',
				body: JSON.stringify({uuid: uuidToUnpin, client_uuid: clientUUID})
			});
			if (!unpinRequest.ok) {
				throw new Error (`Failed to unpin image: ${unpinRequest.status} ${unpinRequest.statusText}`);
			}
			await fetchManifestData(clientUUID as string).then(updatedManifest => {
				container.innerHTML = '';
				renderFiles(updatedManifest, clientUUID as string);
			});
		} catch (unpinError) {
			if (unpinError instanceof Error) {
				alert(`Error unpinning image: ${unpinError.message}`);
			}
		} finally {
			el.disabled = false;
			await initClientImages();
		}
	});

	deleteEl.addEventListener('click', async (event) => {
		const deleteEl = event.currentTarget as HTMLButtonElement;
		if (!(deleteEl instanceof HTMLElement)) return;
		deleteEl.disabled = true;
		const uuidToDelete = deleteEl.dataset.uuid;
		if (!uuidToDelete) {
			alert('Error: No UUID found for this image.');
			deleteEl.disabled = false;
			return;
		}
		const imageCount = deleteEl.dataset.imageCount || '';

		const entry = document.getElementById(uuidToDelete);
		if (entry) {
			entry.style.transition = entry.style.transition || 'opacity 150ms ease';
			entry.style.opacity = '0.5';
			await waitForNextPaint(2);
		}

		const okToDelete = window.confirm(`Are you sure you want to delete this image (${imageCount})?`);
		if (!okToDelete) {
			if (entry) entry.style.opacity = '1';
			deleteEl.disabled = false;
			return;
		}

		try {
			const deleteURL = new URL(`/api/client/files`, window.location.origin);
			deleteURL.searchParams.set('client_uuid', clientUUID);
			deleteURL.searchParams.set('file_uuid', uuidToDelete);
			const deleteResponse = await fetch(deleteURL, {
				method: 'DELETE',
				credentials: 'same-origin'
			});
			if (!deleteResponse.ok) {
				throw new Error (`Failed to delete image: ${deleteResponse.status} ${deleteResponse.statusText}`);
			}
			if (entry) entry.remove();
		} catch (deleteError) {
			if (deleteError instanceof Error) {
				alert(`Error deleting image: ${deleteError.message}`);
			}
		} finally {
			deleteEl.disabled = false;
			await initClientImages();
		}
	});
} 

document.addEventListener('DOMContentLoaded', async () => {
	await initClientImages();
});

async function initClientImages() {
	container.innerHTML = '<p>Loading images...</p>';
	const urlParams = new URLSearchParams(window.location.search);
	const clientUUID = urlParams.get('client_uuid');
	if (!clientUUID) {
		console.warn('No client UUID parameter found in URL.');
		const errorParagraph = document.createElement('p');
		errorParagraph.textContent = `No images found for client UUID: ${clientUUID}`;
		container.appendChild(errorParagraph);
		return;
	}
	if (!clientUUID) {
		console.warn(`Invalid client UUID: ${clientUUID}`);
		return;
	}
	try {
		const manifestData = await fetchManifestData(clientUUID);
		if (manifestData.length === 0) {
			console.warn(`No images found for client UUID: ${clientUUID}`);
			const errorParagraph = document.createElement('p');
			errorParagraph.textContent = `No images found for client UUID: ${clientUUID}`;
			container.appendChild(errorParagraph);
			return;
		}
		renderFiles(manifestData, clientUUID as string);
	} catch (err) {
		container.innerHTML = '';
		const errorParagraph = document.createElement('p');
		if (err instanceof Error) {
			errorParagraph.textContent = `Error fetching images: ${err.message}`;
			container.appendChild(errorParagraph);
			console.warn(`Error fetching images for client UUID ${clientUUID}: ${err.message}`);
		}
	}
}