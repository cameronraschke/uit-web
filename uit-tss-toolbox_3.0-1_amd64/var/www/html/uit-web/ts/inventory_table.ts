const INVENTORY_TABLE_PAGE_SIZE = 20;
let inventoryTableCurrentPageIndex = 0;

function removePortalTooltip() {
	if (activePortalTooltip !== null) {
		activePortalTooltip.remove();
		activePortalTooltip = null;
	}
}

function ensurePortalTooltipListeners() {
	if (hasPortalTooltipGlobalListeners) {
		return;
	}
	hasPortalTooltipGlobalListeners = true;
	window.addEventListener('scroll', removePortalTooltip, true);
	window.addEventListener('resize', removePortalTooltip);
}

function showPortalTooltip(anchor: HTMLElement, text: string) {
	removePortalTooltip();

	const tooltip = document.createElement('div');
	tooltip.classList.add('tooltip-textbox');
	tooltip.textContent = text;
	document.body.appendChild(tooltip);

	const margin = 8;
	const anchorRect = anchor.getBoundingClientRect();
	const tipRect = tooltip.getBoundingClientRect();

	let top = anchorRect.top + window.scrollY - tipRect.height - margin;
	if (top < window.scrollY + margin) {
		top = anchorRect.bottom + window.scrollY + margin;
	}

	let left = anchorRect.left + window.scrollX + (anchorRect.width / 2) - (tipRect.width / 2);
	const minLeft = window.scrollX + margin;
	const maxLeft = window.scrollX + window.innerWidth - tipRect.width - margin;
	if (left < minLeft) {
		left = minLeft;
	}
	if (left > maxLeft) {
		left = maxLeft;
	}

	tooltip.style.top = `${Math.round(top)}px`;
	tooltip.style.left = `${Math.round(left)}px`;
	activePortalTooltip = tooltip;
}

function attachPortalTooltip(anchor: HTMLElement, text: string) {				
	ensurePortalTooltipListeners();
	anchor.addEventListener('mouseenter', () => showPortalTooltip(anchor, text));
	anchor.addEventListener('mouseleave', removePortalTooltip);
	// anchor.addEventListener('focus', () => showPortalTooltip(anchor, text));
	// anchor.addEventListener('blur', removePortalTooltip);
}

function createManufacturerModelCell(inventoryRow: InventoryTableRow) {
  const cell = document.createElement('td');
  
  if (!inventoryRow.system_manufacturer || !inventoryRow.system_model) {
    cell.textContent = 'N/A';
    return cell;
  }
  
  cell.dataset.system_manufacturer = inventoryRow.system_manufacturer;
  cell.dataset.system_model = inventoryRow.system_model;
  
	if (inventoryRow.system_manufacturer === null) {
		inventoryRow.system_manufacturer = 'Unknown Manufacturer';
	} else if (inventoryRow.system_model === null) {
		inventoryRow.system_model = 'Unknown Model';
	}

  const fullText = `${inventoryRow.system_manufacturer}/${inventoryRow.system_model}`;
  
  if (fullText.length > 30) {
    const truncated = `${inventoryRow.system_manufacturer.substring(0, 10)}.../${inventoryRow.system_model.substring(0, 17)}...`;
    cell.textContent = truncated;
    cell.title = fullText;
    cell.style.cursor = 'pointer';
    cell.addEventListener('click', () => {
      cell.textContent = fullText;
    }, { once: true });
  } else {
    cell.textContent = fullText;
  }
  
  return cell;
}

function getInventoryTablePageNumberFromURL() {
	const pageParam = new URLSearchParams(window.location.search).get('page');
	const pageNumber = Number(pageParam);
	if (!Number.isInteger(pageNumber) || pageNumber < 1) {
		return 1;
	}
	return pageNumber;
}

function setInventoryTablePageNumberInURL(pageNumber: number) {
	const nextURL = new URL(window.location.href);
	if (pageNumber <= 1) {
		nextURL.searchParams.delete('page');
	} else {
		nextURL.searchParams.set('page', pageNumber.toString());
	}

	const nextSearch = nextURL.searchParams.toString();
	history.replaceState(null, '', nextSearch ? `${nextURL.pathname}?${nextSearch}` : nextURL.pathname);
}

function getInventoryTablePageHref(pageNumber: number) {
	const nextURL = new URL(window.location.href);
	if (pageNumber <= 1) {
		nextURL.searchParams.delete('page');
	} else {
		nextURL.searchParams.set('page', pageNumber.toString());
	}
	return nextURL.toString();
}

function getInventoryTableSearchQuery() {
	if (!inventoryTableSearch) {
		return {
			searchIncludesSpecialChars: false,
			normalizedSearchText: '',
		};
	}

	let searchIncludesSpecialChars = /[^a-zA-Z0-9]/.test(inventoryTableSearch.value);
	let normalizedSearchText = String(inventoryTableSearch.value.trim().toLowerCase());
	if (normalizedSearchText === '') {
		searchIncludesSpecialChars = false;
	}
	normalizedSearchText = !searchIncludesSpecialChars ? normalizedSearchText.replace(/[^a-zA-Z0-9]/g, '') : normalizedSearchText;

	return {
		searchIncludesSpecialChars,
		normalizedSearchText,
	};
}

function applyInventoryTableSearch(tableData: InventoryTableRow[]) {
	const { searchIncludesSpecialChars, normalizedSearchText } = getInventoryTableSearchQuery();
	if (normalizedSearchText === '') {
		return tableData;
	}

	return tableData.filter((inventoryRow) => {
		let searchableData = `${inventoryRow.tagnumber ?? ''} ${inventoryRow.system_serial ?? ''} ${inventoryRow.location_formatted ?? ''} ${inventoryRow.note ?? ''}`.toLowerCase();
		searchableData = !searchIncludesSpecialChars ? searchableData.replace(/[^a-zA-Z0-9]/g, '') : searchableData;
		return searchableData.includes(normalizedSearchText);
	});
}

function applyInventoryTableSort(tableData: InventoryTableRow[]) {
	const sortedData = [...tableData];
	const selectedSort = inventoryTableSortBy?.value ?? 'time-desc';
	const sortKeys = selectedSort.split('-');
	const sortKey = sortKeys[0] ?? 'time';
	const sortOrder = sortKeys[1] ?? 'desc';

	sortedData.sort((a, b) => {
		if (sortKey === 'time') {
			const aTime = new Date(a.last_updated || 0).getTime();
			const bTime = new Date(b.last_updated || 0).getTime();
			return sortOrder === 'asc' ? aTime - bTime : bTime - aTime;
		}
		if (sortKey === 'tagnumber') {
			return sortOrder === 'asc' ? Number(a.tagnumber) - Number(b.tagnumber) : Number(b.tagnumber) - Number(a.tagnumber);
		}
		if (sortKey === 'serial') {
			const aSerial = a.system_serial?.trim() ?? '';
			const bSerial = b.system_serial?.trim() ?? '';
			return sortOrder === 'asc' ? aSerial.localeCompare(bSerial) : bSerial.localeCompare(aSerial);
		}
		if (sortKey === 'location') {
			const aLocation = a.location_formatted ?? '';
			const bLocation = b.location_formatted ?? '';
			return sortOrder === 'asc' ? aLocation.localeCompare(bLocation) : bLocation.localeCompare(aLocation);
		}
		return 0;
	});

	return sortedData;
}

function renderInventoryPagination(totalRows: number, currentPageIndex: number) {
	if (!inventoryTablePagination) {
		return;
	}

	const pageCount = Math.max(1, Math.ceil(totalRows / INVENTORY_TABLE_PAGE_SIZE));
	if (pageCount <= 1) {
		inventoryTablePagination.innerHTML = '';
		inventoryTablePagination.style.display = 'none';
		return;
	}

	inventoryTablePagination.style.display = 'flex';
	inventoryTablePagination.replaceChildren();

	for (let pageNumber = 1; pageNumber <= pageCount; pageNumber++) {
		const pageAnchor = document.createElement('a');
		pageAnchor.href = getInventoryTablePageHref(pageNumber);
		pageAnchor.textContent = pageNumber.toString();
		pageAnchor.classList.add('inventory-page-link');
		if (pageNumber - 1 === currentPageIndex) {
			pageAnchor.classList.add('active');
		}
		pageAnchor.addEventListener('click', async (event) => {
			event.preventDefault();
			const targetPageIndex = pageNumber - 1;
			if (targetPageIndex === inventoryTableCurrentPageIndex) {
				return;
			}
			inventoryTableCurrentPageIndex = targetPageIndex;
			const renderSucceeded = await renderInventoryTable(
				targetPageIndex * INVENTORY_TABLE_PAGE_SIZE,
				(targetPageIndex + 1) * INVENTORY_TABLE_PAGE_SIZE,
				true,
				false,
			);
			if (renderSucceeded) {
				setInventoryTablePageNumberInURL(pageNumber);
				document.getElementById('update-and-search-container')?.scrollIntoView({ block: 'start', behavior: 'instant', inline: 'nearest' });
			}
		});
		inventoryTablePagination.appendChild(pageAnchor);
	}
}

// Empty table state
function renderEmptyTable(tableBodyEl: HTMLTableSectionElement, message: string) {
	if (!tableBodyEl) return;
  if (inventoryTableRowCountEl) inventoryTableRowCountEl.textContent = '0 entries';
	inventoryTableCurrentPageIndex = 0;
	if (inventoryTablePagination) {
		inventoryTablePagination.innerHTML = '';
		inventoryTablePagination.style.display = 'none';
	}
  tableBodyEl.innerHTML = '';
  
  const inventoryRow = document.createElement('tr');
  const cell = document.createElement('td');
  cell.colSpan = inventoryTable?.rows[0]?.cells.length ?? 1; // span all columns
  cell.textContent = message;
  inventoryRow.appendChild(cell);
  tableBodyEl.appendChild(inventoryRow);
}

async function renderInventoryTable(minimumRowIndex = 0, maximumRowIndex = INVENTORY_TABLE_PAGE_SIZE, skipURLUpdate = false, resolvePageFromURL = true) {
	const isDefaultPageRender = minimumRowIndex === 0 && maximumRowIndex === INVENTORY_TABLE_PAGE_SIZE;
	if (resolvePageFromURL && isDefaultPageRender) {
		const pageNumberFromURL = getInventoryTablePageNumberFromURL();
		minimumRowIndex = (pageNumberFromURL - 1) * INVENTORY_TABLE_PAGE_SIZE;
		maximumRowIndex = minimumRowIndex + INVENTORY_TABLE_PAGE_SIZE;
	}

	if (!skipURLUpdate) {
		updateURLFromAdvFilters(); // necessary, fetchFilteredInventoryData relies on URL parameters
	}
	removePortalTooltip();
	try {
		const tableData: InventoryTableRow[] | null = await fetchFilteredInventoryData();
		if (tableData === null) {
			renderEmptyTable(inventoryTableBody, 'No results found.');
			return false;
		}
		if (!Array.isArray(tableData) || tableData.length === 0) {
			renderEmptyTable(inventoryTableBody, 'No results found.');
			return false;
		}

		const sortedData = applyInventoryTableSort(applyInventoryTableSearch(tableData));
		if (sortedData.length === 0) {
			renderEmptyTable(inventoryTableBody, 'No results found.');
			return false;
		}

		// Row count
		if (inventoryTableRowCountEl) inventoryTableRowCountEl.textContent = `${sortedData.length} entries`;
		inventoryTableCurrentPageIndex = Math.max(0, Math.floor(minimumRowIndex / INVENTORY_TABLE_PAGE_SIZE));

		// Fragment
		const inventoryTableFragment = document.createDocumentFragment();

		// Table body
		const totalRows = sortedData.length;
		if (maximumRowIndex !== 0 && minimumRowIndex >= totalRows && totalRows > 0) {
			inventoryTableCurrentPageIndex = Math.max(0, Math.ceil(totalRows / INVENTORY_TABLE_PAGE_SIZE) - 1);
			minimumRowIndex = inventoryTableCurrentPageIndex * INVENTORY_TABLE_PAGE_SIZE;
			maximumRowIndex = minimumRowIndex + INVENTORY_TABLE_PAGE_SIZE;
		}
		const visibleMaximumRowIndex = maximumRowIndex === 0 ? totalRows : maximumRowIndex;
		const startIndex = Math.max(0, minimumRowIndex);
		const endIndex = Math.min(visibleMaximumRowIndex, totalRows);

		for (let rowIndex = startIndex; rowIndex < endIndex; rowIndex++) {
			const inventoryRow = sortedData[rowIndex];
			if (!inventoryRow) {
				continue;
			}
			
			const tr = document.createElement('tr');

			// variables & dataset values
			const lastUpdated = inventoryRow.last_updated ? new Date(inventoryRow.last_updated).getTime() : '';
			const tagnumber = inventoryRow.tagnumber.toString() ?? '';
			const systemSerial = inventoryRow.system_serial ? inventoryRow.system_serial.trim() : '';
			const locationFormatted = inventoryRow.location_formatted ?? '';
			const departmentFormatted = inventoryRow.department_formatted ?? '';
			const building = inventoryRow.building ?? '';
			const room = inventoryRow.room ?? '';
			const note = inventoryRow.note ?? '';
			const systemManufacturer = inventoryRow.system_manufacturer ?? '';
			const systemModel = inventoryRow.system_model ?? '';
			const deviceTypeFormatted = inventoryRow.device_type_formatted ?? '';
			const adDomainFormatted = inventoryRow.ad_domain_formatted ?? '';
			const osInstalled = Boolean(inventoryRow.os_installed);
			const osName = inventoryRow.os_name ?? '';
			const osVersion = inventoryRow.os_version ?? '';


			tr.dataset.lastUpdated = lastUpdated.toString();
			tr.dataset.tagnumber = tagnumber;
			tr.dataset.systemSerial = systemSerial;
			tr.dataset.locationFormatted = locationFormatted;
			tr.dataset.note = note;
			// tr.dataset.systemManufacturer = systemManufacturer;
			// tr.dataset.systemModel = systemModel;
			// tr.dataset.deviceType = deviceTypeFormatted;
			// tr.dataset.adDomain = adDomainFormatted;
			// tr.dataset.osName = osName;
			// tr.dataset.osVersion = osVersion;

			// Actions cell
			const actionsCell = document.createElement('td');
			actionsCell.classList.add('flex-container', 'horizontal', 'centered');
			const actionsContainer = document.createElement('div');
			const editAnchor = document.createElement('a');
			const editButton = document.createElement('button');
			const imagesAnchor = document.createElement('a');
			const viewImagesButton = document.createElement('button');

			actionsContainer.classList.add('flex-container', 'vertical', 'centered');
			editAnchor.classList.add('smaller-text');
			editButton.classList.add('svg-button', 'text-left', 'edit');
			viewImagesButton.classList.add('svg-button', 'text-left', 'photo-album');

			const editURL = new URL(window.location.href);
			editURL.searchParams.set('tagnumber', tagnumber);
			editURL.searchParams.set('system_serial', systemSerial);
			editURL.searchParams.set('update', 'true');
			editAnchor.href = editURL.toString();

			const imagesURL = new URL(`client_images?tagnumber=${tagnumber}`, window.location.origin);
			imagesAnchor.target = '_blank';
			imagesAnchor.href = imagesURL.toString();
	
			editButton.textContent = 'Edit';

			if (inventoryRow.file_count !== null && inventoryRow.file_count > 0) {
				viewImagesButton.textContent = `Images (${inventoryRow.file_count})`;
			} else {
				viewImagesButton.textContent = 'Images (0)';
				viewImagesButton.style.backgroundColor = "var(--transparent-accent-color)";
			}

			editAnchor.appendChild(editButton);
			imagesAnchor.appendChild(viewImagesButton);
			actionsContainer.appendChild(editAnchor);
			actionsContainer.appendChild(imagesAnchor);

			if (inventoryRow.checkout_bool === true) {
				const printAnchor = document.createElement('a');
				printAnchor.target = '_blank';
				printAnchor.href = new URL(`checkout-form?tagnumber=${tagnumber}`, window.location.origin).toString();

				const printButton = document.createElement('button');
				printButton.classList.add('svg-button', 'text-left', 'print');
				printButton.textContent = 'Checkout Form';
				printAnchor.appendChild(printButton);
				actionsContainer.appendChild(printAnchor);
			}
			
			actionsCell.appendChild(actionsContainer);
			tr.appendChild(actionsCell);

			// Tag Number URL with system serial as well
			const idCell = document.createElement('td');
			const idContainer = document.createElement('div');

			idContainer.classList.add('flex-container', 'vertical', 'centered');
			
			// Tag number
			const tagSpan = document.createElement('span');
			const tagAnchor = document.createElement('a');
			tagAnchor.classList.add('hover-link');
			const tagURL = new URL(`client?tagnumber=${tagnumber}`, window.location.origin);
			tagAnchor.href = tagURL.toString();
			tagAnchor.target = '_blank';
			tagAnchor.appendChild(document.createTextNode(tagnumber));
			tagSpan.appendChild(tagAnchor);
			idContainer.appendChild(tagSpan);

			// System Serial
			const serialSpan = document.createElement('span');
			serialSpan.classList.add('smaller-text');
			serialSpan.textContent = systemSerial;
			idContainer.appendChild(serialSpan);

			idCell.appendChild(idContainer);
			tr.appendChild(idCell);

			// Location Column
			const locationCell = document.createElement('td');
			const locationContainer = document.createElement('div');
			locationContainer.classList.add('flex-container', 'vertical', 'centered');

			// Location
			const locationSpan = document.createElement('span');
			locationSpan.textContent = locationFormatted || 'N/A';
			if (locationFormatted === '') locationSpan.style.fontStyle = 'italic';
			locationContainer.appendChild(locationSpan);

			// Building/room
			const buildingRoomSpan = document.createElement('span');
			buildingRoomSpan.classList.add('smaller-text');
			buildingRoomSpan.textContent = `Bldg: ${building || 'N/A'} - Rm: ${room || 'N/A'}`;
			if (building === '' || room === '') buildingRoomSpan.style.fontStyle = 'italic';
			locationContainer.appendChild(buildingRoomSpan);

			// Department
			const deptSpan = document.createElement('span');
			deptSpan.classList.add('smaller-text');
			const departmentText = document.createTextNode(`Dept: ${departmentFormatted || 'N/A'}`);
			if (departmentFormatted === '') deptSpan.style.fontStyle = 'italic';
			deptSpan.appendChild(departmentText);
			locationContainer.appendChild(deptSpan);

			locationCell.appendChild(locationContainer);
			tr.appendChild(locationCell);

			// Hardware Info
			const hardwareCell = document.createElement('td');

			const hardwareContainer = document.createElement('div');
			hardwareContainer.classList.add('flex-container', 'vertical', 'centered');

			// Manufacturer/Model
			const manufacturerModelSpan = document.createElement('span');
			if (systemManufacturer !== '' && systemModel !== '') {
				manufacturerModelSpan.textContent = `${systemManufacturer}/${systemModel}`;
			} else if (systemManufacturer !== '' && systemModel === '') {
				manufacturerModelSpan.textContent = `${systemManufacturer}/Unknown Model`;
			} else if (systemManufacturer === '' && systemModel !== '') {
				manufacturerModelSpan.textContent = `Unknown Manufacturer/${systemModel}`;
			} else {
				manufacturerModelSpan.style.fontStyle = 'italic';
			}
			if (manufacturerModelSpan.textContent.length > 30) {
				const arr = manufacturerModelSpan.textContent.split('/');
				
				const truncated = (arr[0] !== undefined && arr[1] !== undefined) ? `${arr[0].substring(0, 11)}.../${arr[1].substring(0, 17)}...` : manufacturerModelSpan.textContent;
				manufacturerModelSpan.title = manufacturerModelSpan.textContent;
				manufacturerModelSpan.style.cursor = 'pointer';
				manufacturerModelSpan.textContent = truncated;
				manufacturerModelSpan.addEventListener('click', () => {
					manufacturerModelSpan.textContent = manufacturerModelSpan.title;
					manufacturerModelSpan.style.cursor = 'auto';
				}, { once: true });
			} else {
				manufacturerModelSpan.textContent = manufacturerModelSpan.textContent;
			}
			hardwareContainer.appendChild(manufacturerModelSpan);

			// Device Type
			const deviceTypeSpan = document.createElement('span');
			deviceTypeSpan.classList.add('smaller-text');
			if (inventoryRow.device_type !== '') {
				deviceTypeSpan.textContent = `Type: ${truncateString(deviceTypeFormatted, 20).truncatedString}`;
			} else {
				deviceTypeSpan.style.fontStyle = 'italic';
			}
			hardwareContainer.appendChild(deviceTypeSpan);

			hardwareCell.appendChild(hardwareContainer);
			tr.appendChild(hardwareCell);

			// Software Info
			const softwareCell = document.createElement('td');
			const softwareContainer = document.createElement('div');
			softwareContainer.classList.add('flex-container', 'vertical', 'centered');

			// OS Installed
			const osSpan = document.createElement('span');
			if (osInstalled === true && inventoryRow.disk_removed === false) {
				if (osName !== '' && osVersion !== '') {
					osSpan.textContent = `${inventoryRow.os_name} (${inventoryRow.os_version})`;
				} else if (osName !== '' && osVersion === '') {
					osSpan.textContent = `${inventoryRow.os_name} (vers. N/A)`;
				} else if (osName === '' && osVersion !== '') {
					osSpan.textContent = `OS Installed (vers. ${inventoryRow.os_version})`;
				} else {
					osSpan.textContent = 'Unknown OS Installed';
				}
			} else {
				if (inventoryRow.disk_removed === true) {
					osSpan.textContent = 'Disk Removed - No OS';
				} else {
					osSpan.textContent = 'OS Not Installed';
				}
				osSpan.style.fontStyle = 'italic';
			}
			softwareContainer.appendChild(osSpan);

			// AD Domain
			const domainSpan = document.createElement('span');
			domainSpan.classList.add('smaller-text');
			domainSpan.appendChild(document.createTextNode('AD Domain: '));
			if (adDomainFormatted !== '' && adDomainFormatted !== 'None') {
				domainSpan.appendChild(document.createTextNode(adDomainFormatted));
			} else {
				domainSpan.appendChild(document.createTextNode('N/A'));
				domainSpan.style.fontStyle = 'italic';
			}
			softwareContainer.appendChild(domainSpan);

			softwareCell.appendChild(softwareContainer);
			tr.appendChild(softwareCell);

			// Status
			const statusCell = document.createElement('td');
			const statusContainer = document.createElement('div');
			statusContainer.classList.add('flex-container', 'vertical', 'centered');
			const statusSpan = document.createElement('span');
			if (inventoryRow.status_formatted !== '') {
				statusSpan.textContent = inventoryRow.status_formatted;
				if (inventoryRow.status === 'retired') {
					const tooltipIndicator = document.createElement('img');
					tooltipIndicator.src = '/icons/general/info.svg';
					tooltipIndicator.classList.add('tooltip-image', 'info');
					tooltipIndicator.setAttribute('tabindex', '0');
					statusSpan.appendChild(tooltipIndicator);
					attachPortalTooltip(
						tooltipIndicator,
						`Retired Date: ${inventoryRow.retired_date ? new Date(inventoryRow.retired_date).toLocaleDateString() : 'N/A'}`,
					);
				}
			} else {
				statusSpan.textContent = 'N/A';
				statusSpan.style.fontStyle = 'italic';
			}
			statusContainer.appendChild(statusSpan);
			statusCell.appendChild(statusContainer);
			tr.appendChild(statusCell);

			// Note (truncated)
			const noteCell = document.createElement('td');
			const noteContainer = document.createElement('div');
			noteContainer.classList.add('flex-container', 'vertical', 'centered');
			if (note !== '') {
				const noteSpan = document.createElement('span');
				const truncatedButtonEl = document.createElement('button');
				const truncatedNote = truncateString(note, 50);
				
				const updateNoteTruncation = () => {
					const isTruncated = noteSpan.dataset.truncated === 'true';
					if (noteContainer.contains(truncatedButtonEl)) {
						noteContainer.removeChild(truncatedButtonEl);
					}
					truncatedButtonEl.removeEventListener('click', () => {}); // remove all listeners to prevent duplicates
					noteSpan.removeEventListener('click', () => {}); // remove all listeners to prevent duplicates
					if (isTruncated) {
						if (noteSpan.textContent !== truncatedNote.truncatedString) {
							noteSpan.textContent = truncatedNote.truncatedString;
							noteSpan.title = `(Click to expand) ${note}`;
							noteSpan.style.cursor = 'pointer';
						}
						addNoteTextListener();
					} else {
						if (noteSpan.textContent !== note) {
							noteSpan.textContent = note;
						}
						noteSpan.removeAttribute('title');
						noteSpan.style.cursor = 'auto';
						if (!noteContainer.contains(truncatedButtonEl)) {
							truncatedButtonEl.classList.add('svg-button', 'small-x');
							truncatedButtonEl.title = 'Collapse note';
							noteContainer.appendChild(truncatedButtonEl);
						}
						addNoteCollapseButtonListener();
					}
				}

				const addNoteCollapseButtonListener = () => {
					truncatedButtonEl.addEventListener('click', (e) => {
						e.stopPropagation();
						noteSpan.dataset.truncated = 'true';
						if (e.currentTarget) {
							e.currentTarget.removeEventListener('click', () => {}); // remove all listeners to prevent duplicates
						}
						updateNoteTruncation();
					});
				};

				const addNoteTextListener = () => {
					noteSpan.addEventListener('click', (e) => {
						e.stopPropagation();
						noteSpan.dataset.truncated = 'false';
						if (e.currentTarget) {
							e.currentTarget.removeEventListener('click', () => {}); // remove all listeners to prevent duplicates
						}
						updateNoteTruncation();
					});
				};
				
				if (truncatedNote.isTruncated) {
					noteSpan.dataset.truncated = 'true';
					updateNoteTruncation();
				} else {
					noteSpan.textContent = note;
					noteSpan.dataset.truncated = 'false';
				}

				noteContainer.appendChild(noteSpan);
			}
			noteCell.appendChild(noteContainer);
			tr.appendChild(noteCell);

			// Last Updated
			const lastUpdatedCell = document.createElement('td');
			const lastUpdatedDiv = document.createElement('div');
			lastUpdatedDiv.classList.add('flex-container', 'vertical');
			const dateFormattedDiv = document.createElement('div');
			dateFormattedDiv.appendChild(document.createTextNode(new Date(inventoryRow.last_updated).toLocaleDateString()));
			const timeFormattedDiv = document.createElement('div');
			timeFormattedDiv.classList.add('flex-container', 'horizontal', 'smaller-text');
			timeFormattedDiv.appendChild(document.createTextNode(new Date(inventoryRow.last_updated).toLocaleTimeString()));
			lastUpdatedDiv.appendChild(dateFormattedDiv);
			lastUpdatedDiv.appendChild(timeFormattedDiv);
			lastUpdatedCell.appendChild(lastUpdatedDiv);
			tr.appendChild(lastUpdatedCell);

			// Tooltip to show errors
			if (inventoryRow.client_configuration_errors && inventoryRow.client_configuration_errors.length > 0) {
				const generalTooltip: HTMLImageElement = document.createElement('img');
				const softwareTooltip: HTMLImageElement = document.createElement('img');

				const hardwareErrArr: Array<string> = [];
				const firmwareErrArr: Array<string> = [];
				const softwareErrArr: Array<string> = [];
				const otherSoftwareErrArr: Array<string> = [];

				const tooltipArr = [generalTooltip, softwareTooltip];
				for (const tooltip of tooltipArr) {
					tooltip.title = 'Configuration Error(s)';
					tooltip.alt = 'Configuration Error(s)';
					tooltip.src = '/icons/general/info.svg';
					tooltip.setAttribute('tabindex', '0');

					let highestGeneralTooltipSeverity = 'info';
					let highestSoftwareTooltipSeverity = 'info';
					for (const err of inventoryRow.client_configuration_errors) {
						// separate tooltips by error type
						if (err.error_type === 'firmware' || err.error_type === 'software') {
							if (err.error_level === 'error' || highestSoftwareTooltipSeverity === 'error') {
								highestSoftwareTooltipSeverity = 'error';
								continue;
							} else if (err.error_level === 'warning' && highestSoftwareTooltipSeverity !== 'error') {
								highestSoftwareTooltipSeverity = 'warning';
							} else if (err.error_level === 'info' && highestSoftwareTooltipSeverity !== 'error' && highestSoftwareTooltipSeverity !== 'warning') {
								highestSoftwareTooltipSeverity = 'info';
							}
						} else {
							if (err.error_level === 'error' || highestGeneralTooltipSeverity === 'error') {
								highestGeneralTooltipSeverity = 'error';
								continue;
							} else if (err.error_level === 'warning' && highestGeneralTooltipSeverity !== 'error') {
								highestGeneralTooltipSeverity = 'warning';
							} else if (err.error_level === 'info' && highestGeneralTooltipSeverity !== 'error' && highestGeneralTooltipSeverity !== 'warning') {
								highestGeneralTooltipSeverity = 'info';
							}
						}
					}
					if (highestGeneralTooltipSeverity === 'error') {
						generalTooltip.classList.add('tooltip-image', 'error');
					} else if (highestGeneralTooltipSeverity === 'warning') {
						generalTooltip.classList.add('tooltip-image', 'warning');
					} else if (highestGeneralTooltipSeverity === 'info') {
						generalTooltip.classList.add('tooltip-image', 'info');
					}

					if (highestSoftwareTooltipSeverity === 'error') {
						softwareTooltip.classList.add('tooltip-image', 'error');
					} else if (highestSoftwareTooltipSeverity === 'warning') {
						softwareTooltip.classList.add('tooltip-image', 'warning');
					}	else if (highestSoftwareTooltipSeverity === 'info') {
						softwareTooltip.classList.add('tooltip-image', 'info');
					}
				}
				for (const err of inventoryRow.client_configuration_errors) {
					if (err.error_type === 'hardware') {
						hardwareErrArr.push(err.error_message);
					} else if (err.error_type === 'firmware') {
						firmwareErrArr.push(err.error_message);
					} else if (err.error_type === 'software') {
						softwareErrArr.push(err.error_message);
					} else if (err.error_type === 'other') {
						otherSoftwareErrArr.push(err.error_message);
					}
				}


				if (hardwareErrArr.length > 0 || otherSoftwareErrArr.length > 0) {
					attachPortalTooltip(
						generalTooltip,
						`Hardware Configuration Error(s): ${(hardwareErrArr.length > 0 ? hardwareErrArr.join(', ') : '') + (otherSoftwareErrArr.length > 0 ? (hardwareErrArr.length > 0 ? ', ' : '') + otherSoftwareErrArr.join(', ') : '')}`,
					);
					tagSpan.appendChild(generalTooltip);
				}
				if (softwareErrArr.length > 0 || firmwareErrArr.length > 0) {
					attachPortalTooltip(
						softwareTooltip,
						`Software Configuration Error(s): ${(softwareErrArr.length > 0 ? softwareErrArr.join(', ') : '') + (firmwareErrArr.length > 0 ? (softwareErrArr.length > 0 ? ', ' : '') + firmwareErrArr.join(', ') : '')}`,
					);
					osSpan.appendChild(softwareTooltip);
				}
			}

			inventoryTableFragment.appendChild(tr);
		}
		if (inventoryTableBody) {
			inventoryTableBody.replaceChildren(inventoryTableFragment);
		}
		setInventoryTablePageNumberInURL(inventoryTableCurrentPageIndex + 1);
		renderInventoryPagination(totalRows, inventoryTableCurrentPageIndex);
		return true;
	} catch (error) {
		console.error('Error rendering inventory table:', error);
		renderEmptyTable(inventoryTableBody, 'Error loading inventory data. Please try again.');
		return false;
	}
}

if (inventoryTableSortBy) {
	inventoryTableSortBy.addEventListener('change', async () => {
		await renderInventoryTable(0, INVENTORY_TABLE_PAGE_SIZE, true, false);
	});
}

if (inventoryTableSearch) {
	inventoryTableSearch.addEventListener('keyup', () => {
		if (inventoryTableSearchDebounce !== null) {
			clearTimeout(inventoryTableSearchDebounce);
		}

		inventoryTableSearchDebounce = setTimeout(() => {
			renderInventoryTable(0, INVENTORY_TABLE_PAGE_SIZE, true, false);
		}, 100);
	});
}