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
	tooltip.classList.add('portal-tooltip');
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
function renderEmptyTable(tableBody: HTMLElement, message: string) {
  rowCountElement.textContent = '0 entries';
  tableBody.innerHTML = '';
  
  const inventoryRow = document.createElement('tr');
  const cell = document.createElement('td');
  cell.colSpan = 10;
  cell.textContent = message;
  inventoryRow.appendChild(cell);
  tableBody.appendChild(inventoryRow);
}

async function renderInventoryTable() {
	updateURLFromFilters(); // necessary, fetchFilteredInventoryData relies on URL parameters
	removePortalTooltip();
	try {
		const tableData: InventoryTableRow[] | null = await fetchFilteredInventoryData();
		if (tableData === null) {
			renderEmptyTable(tableBody, 'No results found.');
			return;
		}
		if (!Array.isArray(tableData) || tableData.length === 0) {
			renderEmptyTable(tableBody, 'No results found.');
			return;
		}

		const sortedData = [...tableData].sort((a, b) => 
			new Date(b.last_updated || 0).getTime() - new Date(a.last_updated || 0).getTime()
		);

		// Row count
		rowCountElement.textContent = `${sortedData.length} entries`;

		// Fragment
		const fragment = document.createDocumentFragment();

		// Table body
		for (const inventoryRow of sortedData) {
			const tr = document.createElement('tr');

			// variables & dataset values
			const lastUpdated = inventoryRow.last_updated ? new Date(inventoryRow.last_updated).getTime() : '';
			const tagnumber = inventoryRow.tagnumber.toString() || '';
			const systemSerial = inventoryRow.system_serial ? inventoryRow.system_serial.trim() : '';
			const locationFormatted = inventoryRow.location_formatted || '';
			const building = inventoryRow.building || '';
			const room = inventoryRow.room || '';
			const note = inventoryRow.note || '';

			tr.dataset.lastUpdated = lastUpdated.toString();
			tr.dataset.tagnumber = tagnumber;
			tr.dataset.systemSerial = systemSerial;
			tr.dataset.locationFormatted = locationFormatted;
			tr.dataset.note = note;

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

			viewImagesButton.textContent = `Images (${inventoryRow.file_count || 0})`;

			tagAnchor.appendChild(editButton);
			imagesAnchor.appendChild(viewImagesButton);
			actionsContainer.appendChild(tagAnchor);
			actionsContainer.appendChild(imagesAnchor);
			actionsCell.appendChild(actionsContainer);
			tr.appendChild(actionsCell);

			// Tag Number URL with system serial as well
			const idCell = document.createElement('td');
			const idContainer = document.createElement('div');
			const tagDiv = document.createElement('div');
			const serialDiv = document.createElement('div');

			idContainer.classList.add('flex-container', 'vertical');
			tagDiv.classList.add('flex-container', 'horizontal');
			tagDiv.style.justifyContent = 'center';
			serialDiv.classList.add('flex-container', 'horizontal', 'smaller-text');
			
			// Tag number
			const tagSpan = document.createElement('span');
			tagSpan.appendChild(document.createTextNode(tagnumber));
			tagDiv.appendChild(tagSpan);
			// Tooltip to show errors
			if (inventoryRow.client_configuration_errors && inventoryRow.client_configuration_errors.length > 0) {
				const tooltipIndicator = document.createElement('img');
				tooltipIndicator.title = 'Configuration Error(s)';
				tooltipIndicator.alt = 'Configuration Error(s)';
				tooltipIndicator.src = '/icons/general/info.svg';
				tooltipIndicator.classList.add('tooltip-image');
				tooltipIndicator.tabIndex = 0;
				attachPortalTooltip(
					tooltipIndicator,
					'Configuration Error(s): ' + inventoryRow.client_configuration_errors.join(', '),
				);
				tagDiv.appendChild(tooltipIndicator);
			}
			serialDiv.textContent = systemSerial;

			idContainer.appendChild(tagDiv);
			idContainer.appendChild(serialDiv);
			idCell.appendChild(idContainer);
			tr.appendChild(idCell);

			// Location
			const locationCell = document.createElement('td');
			const locationContainer = document.createElement('div');
			const locationFormattedDiv = document.createElement('div');
			const buildingRoomDiv = document.createElement('div');
			const departmentDiv = document.createElement('div');

			locationContainer.classList.add('flex-container', 'vertical');
			locationFormattedDiv.classList.add('flex-container', 'horizontal');
			buildingRoomDiv.classList.add('flex-container', 'horizontal', 'smaller-text');
			departmentDiv.classList.add('flex-container', 'horizontal', 'smaller-text');

			locationFormattedDiv.textContent = locationFormatted || 'N/A';
			buildingRoomDiv.textContent = `B: ${building || 'N/A'} - R: ${room || 'N/A'}`;
			if (!locationFormatted) locationFormattedDiv.style.fontStyle = 'italic';
			if (!building && !room) buildingRoomDiv.style.fontStyle = 'italic';
			if (!inventoryRow.department_formatted) departmentDiv.style.fontStyle = 'italic';
			departmentDiv.appendChild(document.createTextNode(inventoryRow.department_formatted || 'N/A'));

			locationContainer.appendChild(locationFormattedDiv);
			locationContainer.appendChild(buildingRoomDiv);
			locationContainer.appendChild(departmentDiv);
			locationCell.appendChild(locationContainer);
			tr.appendChild(locationCell);

			// Manufacturer and Model combined cell
			const manufacturerModelCell = document.createElement('td');
			const manufacturerModelContainer = document.createElement('div');
			const deviceTypeContainer = document.createElement('div');
			const manufacturerModelDiv = document.createElement('div');

			manufacturerModelContainer.classList.add('flex-container', 'vertical');
			deviceTypeContainer.classList.add('flex-container', 'horizontal');
			manufacturerModelDiv.classList.add('flex-container', 'horizontal', 'smaller-text');

			manufacturerModelCell.dataset.systemManufacturer = inventoryRow.system_manufacturer || '';
			manufacturerModelCell.dataset.systemModel = inventoryRow.system_model || '';
			manufacturerModelCell.dataset.deviceType = inventoryRow.device_type || '';


			let deviceTypeText = 'N/A';
			if (inventoryRow.device_type) {
				deviceTypeText = inventoryRow.device_type_formatted || inventoryRow.device_type;
			} else {
				deviceTypeContainer.style.fontStyle = 'italic';
			}
			deviceTypeContainer.textContent = truncateString(deviceTypeText, 20).truncatedString;

			let manufacturerModelText = 'N/A';
			if (inventoryRow.system_manufacturer && inventoryRow.system_model) {
				manufacturerModelText = `${inventoryRow.system_manufacturer}/${inventoryRow.system_model}`;
			} else if (inventoryRow.system_manufacturer && !inventoryRow.system_model) {
				manufacturerModelText = `${inventoryRow.system_manufacturer}/Unknown Model`;
			} else if (!inventoryRow.system_manufacturer && inventoryRow.system_model) {
				manufacturerModelText = `Unknown Manufacturer/${inventoryRow.system_model}`;
			} else {
				manufacturerModelDiv.style.fontStyle = 'italic';
			}
			if (manufacturerModelText.length > 30) {
				const arr = manufacturerModelText.split('/');
				const truncated = `${arr[0].substring(0, 11)}.../${arr[1].substring(0, 17)}...`;
				manufacturerModelDiv.title = manufacturerModelText;
				manufacturerModelDiv.style.cursor = 'pointer';
				manufacturerModelDiv.textContent = truncated;
				manufacturerModelDiv.addEventListener('click', () => {
					manufacturerModelDiv.textContent = manufacturerModelText;
					manufacturerModelDiv.style.cursor = 'auto';
				}, { once: true });
			} else {
				manufacturerModelDiv.textContent = manufacturerModelText;
			}

			manufacturerModelContainer.appendChild(deviceTypeContainer)
			manufacturerModelContainer.appendChild(manufacturerModelDiv);
			manufacturerModelCell.appendChild(manufacturerModelContainer);
			tr.appendChild(manufacturerModelCell);
			

			// AD Domain
			tr.appendChild(createTextCell(undefined, 'ad_domain', inventoryRow.ad_domain_formatted, 20, undefined));

			// Status
			tr.appendChild(createTextCell(undefined, 'status', inventoryRow.status_formatted, undefined, undefined));

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
		tableBody.replaceChildren(fragment);
	} catch (error) {
		console.error('Error rendering inventory table:', error);
		renderEmptyTable(tableBody, 'Error loading inventory data. Please try again.');
	}
}

inventoryTableSortBy.addEventListener('change', () => {
	const presentRows = Array.from(tableBody.querySelectorAll('tr')).filter(row => row.style.display !== 'none');
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
	sortedRows.forEach(row => tableBody.appendChild(row.rowElement));
});

inventoryTableSearch.addEventListener('keyup', () => {
	if (inventoryTableSearchDebounce !== null) {
		clearTimeout(inventoryTableSearchDebounce);
	}

	inventoryTableSearchDebounce = setTimeout(() => {
		let searchIncludesSpecialChars = /[^a-zA-Z0-9]/.test(inventoryTableSearch.value);

		let lowerCaseSearchedTextInput = String(inventoryTableSearch.value.trim().toLowerCase());
		if (lowerCaseSearchedTextInput === '') searchIncludesSpecialChars = false;
		lowerCaseSearchedTextInput = !searchIncludesSpecialChars ? lowerCaseSearchedTextInput.replace(/[^a-zA-Z0-9]/g, "") : lowerCaseSearchedTextInput;
		const allRows = Array.from(tableBody.querySelectorAll('tr'));
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
		rowCountElement.textContent = `${allRows.filter(row => row.style.display === 'table-row').length} entries`;
	}, 100);
});