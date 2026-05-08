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

// Empty table state
function renderEmptyTable(tableBodyEl: HTMLTableSectionElement, message: string) {
	if (!tableBodyEl) return;
  if (inventoryTableRowCountEl) inventoryTableRowCountEl.textContent = '0 entries';
  tableBodyEl.innerHTML = '';
  
  const inventoryRow = document.createElement('tr');
  const cell = document.createElement('td');
  cell.colSpan = 10;
  cell.textContent = message;
  inventoryRow.appendChild(cell);
  tableBodyEl.appendChild(inventoryRow);
}

async function renderInventoryTable() {
	updateURLFromAdvFilters(); // necessary, fetchFilteredInventoryData relies on URL parameters
	removePortalTooltip();
	try {
		const tableData: InventoryTableRow[] | null = await fetchFilteredInventoryData();
		if (tableData === null) {
			renderEmptyTable(inventoryTableBody, 'No results found.');
			return;
		}
		if (!Array.isArray(tableData) || tableData.length === 0) {
			renderEmptyTable(inventoryTableBody, 'No results found.');
			return;
		}

		const sortedData = [...tableData].sort((a, b) => 
			new Date(b.last_updated || 0).getTime() - new Date(a.last_updated || 0).getTime()
		);

		// Row count
		if (inventoryTableRowCountEl) inventoryTableRowCountEl.textContent = `${sortedData.length} entries`;

		// Fragment
		const fragment = document.createDocumentFragment();

		// Table body
		for (const inventoryRow of sortedData) {
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
			const actionsContainer = document.createElement('div');
			const tagAnchor = document.createElement('a');
			const editButton = document.createElement('button');
			const imagesAnchor = document.createElement('a');
			const viewImagesButton = document.createElement('button');

			actionsContainer.classList.add('flex-container', 'horizontal');
			tagAnchor.classList.add('smaller-text');
			editButton.classList.add('svg-button', 'edit');
			viewImagesButton.classList.add('svg-button', 'photo-album');

			const tagURL = new URL(window.location.href);
			tagURL.searchParams.set('tagnumber', tagnumber);
			tagURL.searchParams.set('system_serial', systemSerial);
			tagURL.searchParams.set('update', 'true');
			tagAnchor.href = tagURL.toString();

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

			tagAnchor.appendChild(editButton);
			imagesAnchor.appendChild(viewImagesButton);
			actionsContainer.appendChild(tagAnchor);
			actionsContainer.appendChild(imagesAnchor);

			if (inventoryRow.checkout_bool === true) {
				const printAnchor = document.createElement('a');
				printAnchor.target = '_blank';
				printAnchor.href = new URL(`checkout-form?tagnumber=${tagnumber}`, window.location.origin).toString();

				const printButton = document.createElement('button');
				printButton.classList.add('svg-button', 'print');
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
			tagSpan.appendChild(document.createTextNode(tagnumber));
			// Tooltip to show errors
			if (inventoryRow.client_configuration_errors && inventoryRow.client_configuration_errors.length > 0) {
				const tooltipIndicator = document.createElement('img');
				tooltipIndicator.title = 'Configuration Error(s)';
				tooltipIndicator.alt = 'Configuration Error(s)';
				tooltipIndicator.src = '/icons/general/info.svg';
				tooltipIndicator.classList.add('tooltip-image', 'error');
				tooltipIndicator.tabIndex = 0;
				attachPortalTooltip(
					tooltipIndicator,
					'Configuration Error(s): ' + inventoryRow.client_configuration_errors.join(', '),
				);
				tagSpan.appendChild(tooltipIndicator);
			}
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
				const truncated = `${arr[0].substring(0, 11)}.../${arr[1].substring(0, 17)}...`;
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
					tooltipIndicator.tabIndex = 0;
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
			tr.appendChild(createTextCell(undefined, 'note', inventoryRow.note, 60, ''));

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

			fragment.appendChild(tr);
		}
		if (inventoryTableBody) inventoryTableBody.replaceChildren(fragment);
	} catch (error) {
		console.error('Error rendering inventory table:', error);
		renderEmptyTable(inventoryTableBody, 'Error loading inventory data. Please try again.');
	}
}

if (inventoryTableSortBy) {
	inventoryTableSortBy.addEventListener('change', () => {
		const presentRows = Array.from(inventoryTableBody.querySelectorAll('tr')).filter(row => row.style.display !== 'none');
		const rowData = presentRows.map(row => ({
			lastUpdated: row.dataset.lastUpdated || '',
			tagnumber: row.dataset.tagnumber || '',
			systemSerial: row.dataset.systemSerial || '',
			locationFormatted: row.dataset.locationFormatted || '',
			rowElement: row
		}));
		const sortedRows = rowData.sort((a, b) => {
			if (!inventoryTableSortBy.value) return 0;
			const sortKeys = inventoryTableSortBy.value.split('-');
			const sortKey = sortKeys[0];
			const sortOrder = sortKeys[1];
			
			if (sortKey === 'time') {
				const aTime = Number(a.lastUpdated) || 0;
				const bTime = Number(b.lastUpdated) || 0;
				return sortOrder === 'asc' ? aTime - bTime : bTime - aTime;
			}
			if (sortKey === 'tagnumber') {
				return sortOrder === 'asc' ? Number(a.tagnumber) - Number(b.tagnumber) : Number(b.tagnumber) - Number(a.tagnumber);
			}
			if (sortKey === 'serial') {
				const aSerial = a.systemSerial || '';
				const bSerial = b.systemSerial || '';
				return sortOrder === 'asc' ? aSerial.localeCompare(bSerial) : bSerial.localeCompare(aSerial);
			}
			if (sortKey === 'location') {
				const aLocation = a.locationFormatted || '';
				const bLocation = b.locationFormatted || '';
				return sortOrder === 'asc' ? aLocation.localeCompare(bLocation) : bLocation.localeCompare(aLocation);
			}
			return 0;
		});
		sortedRows.forEach(row => inventoryTableBody.appendChild(row.rowElement));
	});
}

if (inventoryTableSearch) {
	inventoryTableSearch.addEventListener('keyup', () => {
		if (inventoryTableSearchDebounce !== null) {
			clearTimeout(inventoryTableSearchDebounce);
		}

		inventoryTableSearchDebounce = setTimeout(() => {
			let searchIncludesSpecialChars = /[^a-zA-Z0-9]/.test(inventoryTableSearch.value);

			let lowerCaseSearchedTextInput = String(inventoryTableSearch.value.trim().toLowerCase());
			if (lowerCaseSearchedTextInput === '') searchIncludesSpecialChars = false;
			lowerCaseSearchedTextInput = !searchIncludesSpecialChars ? lowerCaseSearchedTextInput.replace(/[^a-zA-Z0-9]/g, "") : lowerCaseSearchedTextInput;
			const allRows = Array.from(inventoryTableBody.querySelectorAll('tr'));
			for (const row of allRows) {
				if (lowerCaseSearchedTextInput === '') {
					row.style.display = 'table-row';
					continue;
				}
				let lowerCaseSearchableData = (row.dataset.tagnumber + ' ' + row.dataset.systemSerial + ' ' + row.dataset.locationFormatted + ' ' + row.dataset.note).toLowerCase();
				lowerCaseSearchableData = !searchIncludesSpecialChars ? lowerCaseSearchableData.replace(/[^a-zA-Z0-9]/g, "") : lowerCaseSearchableData;
				if (lowerCaseSearchableData.includes(lowerCaseSearchedTextInput)) { // lowerCaseSearchedTextInput values are already lower case
					row.style.display = 'table-row';
				} else {
					row.style.display = 'none';
				}
			}
			if (inventoryTableRowCountEl) inventoryTableRowCountEl.textContent = `${allRows.filter(row => row.style.display === 'table-row').length} entries`;
		}, 100);
	});
}