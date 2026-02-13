type InventoryRow = {
	tagnumber: number | 0;
	system_serial: string | "";
	location_formatted: string | "";
	building: string | "";
	room: string | "";
	system_manufacturer: string | "";
	system_model: string | "";
	device_type: string | "";
	device_type_formatted: string | "";
	department_formatted: string | "";
	ad_domain_formatted: string | "";
	status: string | "";
	is_broken: boolean | null;
	note: string | "";
	last_updated: string | "";
};

// Table elements
const tableBody = document.getElementById('inventory-table-body') as HTMLElement;
const rowCountElement = document.getElementById('inventory-table-rowcount') as HTMLElement;
const inventoryTableSearch = document.getElementById('inventory-table-search') as HTMLInputElement;
const inventoryTableSortBy = document.getElementById('inventory-table-sort-by') as HTMLSelectElement;
let inventoryTableSearchDebounce: ReturnType<typeof setTimeout> | null = null;

function createManufacturerModelCell(jsonRow: InventoryRow) {
  const cell = document.createElement('td');
  
  if (!jsonRow.system_manufacturer || !jsonRow.system_model) {
    cell.textContent = 'N/A';
    return cell;
  }
  
  cell.dataset.system_manufacturer = jsonRow.system_manufacturer;
  cell.dataset.system_model = jsonRow.system_model;
  
	if (jsonRow.system_manufacturer === null) {
		jsonRow.system_manufacturer = 'Unknown Manufacturer';
	} else if (jsonRow.system_model === null) {
		jsonRow.system_model = 'Unknown Model';
	}

  const fullText = `${jsonRow.system_manufacturer}/${jsonRow.system_model}`;
  
  if (fullText.length > 30) {
    const truncated = `${jsonRow.system_manufacturer.substring(0, 10)}.../${jsonRow.system_model.substring(0, 17)}...`;
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
  
  const jsonRow = document.createElement('tr');
  const cell = document.createElement('td');
  cell.colSpan = 10;
  cell.textContent = message;
  jsonRow.appendChild(cell);
  tableBody.appendChild(jsonRow);
}

async function renderInventoryTable() {
	updateURLFromFilters(); // necessary, fetchFilteredInventoryData relies on URL parameters
	try {
		const tableData = await fetchFilteredInventoryData();
		if (tableData === null) {
			renderEmptyTable(tableBody, 'Error loading inventory data. Please try again.');
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
		for (const jsonRow of sortedData) {
			const tr = document.createElement('tr');

			// variables & dataset values
			const lastUpdated = jsonRow.last_updated ? new Date(jsonRow.last_updated).getTime() : '';
			const tagnumber = jsonRow.tagnumber.toString() || '';
			const systemSerial = jsonRow.system_serial ? jsonRow.system_serial.trim() : '';
			const locationFormatted = jsonRow.location_formatted || '';
			const building = jsonRow.building || '';
			const room = jsonRow.room || '';
			const note = jsonRow.note || '';

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
			viewImagesButton.textContent = 'Images';

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
			serialDiv.classList.add('flex-container', 'horizontal', 'smaller-text');
			
			tagDiv.textContent = tagnumber;
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

			locationContainer.classList.add('flex-container', 'vertical');
			locationFormattedDiv.classList.add('flex-container', 'horizontal');
			buildingRoomDiv.classList.add('flex-container', 'horizontal', 'smaller-text');

			locationFormattedDiv.textContent = locationFormatted || 'N/A';
			buildingRoomDiv.textContent = `B: ${building || 'N/A'} - R: ${room || 'N/A'}`;
			if (!locationFormatted) locationFormattedDiv.style.fontStyle = 'italic';
			if (!building && !room) buildingRoomDiv.style.fontStyle = 'italic';

			locationContainer.appendChild(locationFormattedDiv);
			locationContainer.appendChild(buildingRoomDiv);
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

			manufacturerModelCell.dataset.systemManufacturer = jsonRow.system_manufacturer || '';
			manufacturerModelCell.dataset.systemModel = jsonRow.system_model || '';
			manufacturerModelCell.dataset.deviceType = jsonRow.device_type || '';


			let deviceTypeText = 'N/A';
			if (jsonRow.device_type) {
				deviceTypeText = jsonRow.device_type_formatted || jsonRow.device_type;
			} else {
				deviceTypeContainer.style.fontStyle = 'italic';
			}
			deviceTypeContainer.textContent = truncateString(deviceTypeText, 20).truncatedString;

			let manufacturerModelText = 'N/A';
			if (jsonRow.system_manufacturer && jsonRow.system_model) {
				manufacturerModelText = `${jsonRow.system_manufacturer}/${jsonRow.system_model}`;
			} else if (jsonRow.system_manufacturer && !jsonRow.system_model) {
				manufacturerModelText = `${jsonRow.system_manufacturer}/Unknown Model`;
			} else if (!jsonRow.system_manufacturer && jsonRow.system_model) {
				manufacturerModelText = `Unknown Manufacturer/${jsonRow.system_model}`;
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

			// Department
			tr.appendChild(createTextCell(undefined, 'department', jsonRow.department_formatted, 20, undefined));

			// Domain
			tr.appendChild(createTextCell(undefined, 'ad_domain', jsonRow.ad_domain_formatted, 20, undefined));

			// Status
			tr.appendChild(createTextCell(undefined, 'status', jsonRow.status, undefined, undefined));

			// Note (truncated)
			tr.appendChild(createTextCell(undefined, 'note', jsonRow.note, 60, ''));

			// Last Updated
			tr.appendChild(createTimestampCell(undefined, 'last_updated', jsonRow.last_updated, undefined));

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