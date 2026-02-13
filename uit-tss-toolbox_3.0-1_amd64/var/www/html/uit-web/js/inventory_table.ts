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

const tagLookupInput = document.getElementById('inventory-tag-lookup') as HTMLInputElement;
const serialLookupInput = document.getElementById('inventory-serial-lookup') as HTMLInputElement;
const rowCountElement = document.getElementById('inventory-table-rowcount') as HTMLElement;
const formAnchor = document.querySelector('#update-and-search-container') as HTMLElement;
const inventoryTagSortInput = document.getElementById('inventory-sort-tagnumber') as HTMLInputElement;
const inventorySerialSortInput = document.getElementById('inventory-sort-serial') as HTMLInputElement;
const inventoryTimeSortInput = document.getElementById('inventory-sort-time') as HTMLInputElement;
const inventorySortByInput = document.getElementById('inventory-sort-by') as HTMLSelectElement;



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
			const tagnumber = jsonRow.tagnumber.toString() || '';
			const systemSerial = jsonRow.system_serial ? jsonRow.system_serial.trim() : '';
			const locationFormatted = jsonRow.location_formatted || '';
			const building = jsonRow.building || '';
			const room = jsonRow.room || '';

			tr.dataset.tagnumber = tagnumber;
			tr.dataset.systemSerial = systemSerial;

			// Actions cell
			const actionsCell = document.createElement('td');
			const actionsContainer = document.createElement('div');
			const tagAnchor = document.createElement('a');
			const editButton = document.createElement('button');

			actionsContainer.classList.add('flex-container', 'horizontal');
			tagAnchor.classList.add('unstyled', 'smaller-text');
			editButton.classList.add('svg-button', 'edit');

			const tagURL = new URL(window.location.href);
			tagURL.searchParams.set('tagnumber', tagnumber);
			tagURL.searchParams.set('system_serial', systemSerial);
			tagURL.searchParams.set('update', 'true');
			tagAnchor.href = tagURL.toString();
			
			editButton.textContent = 'Edit';

			tagAnchor.appendChild(editButton);
			actionsContainer.appendChild(tagAnchor);
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
				deviceTypeText = jsonRow.device_type;
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

function getInventorySortByParams() {
	const sortBy = inventorySortByInput.value.trim();
	const sortByArr = sortBy.split('-');
	const sortKey = sortByArr[0];
	const sortOrder = sortByArr[1];
	if (sortKey.trim() === '' || sortOrder.trim() === '') {
		return null;
	}
	const table = document.getElementById('inventory-table') as HTMLTableElement;
	const tbody = table.querySelector("tbody") as HTMLTableSectionElement;
	if (!table || !tbody) {
		return null;
	}
	return { sortKey, sortOrder, table, tbody };
}

function sortInventoryTable(sortKey: string, sortOrder: string, tbody: HTMLTableSectionElement) {
	const rowsArray = Array.from(tbody.rows);
	rowsArray.sort((a, b) => {
		const aValue = a.dataset[sortKey] || '';
		const bValue = b.dataset[sortKey] || '';
		if (sortKey === 'last_updated') {
			const aDate = new Date(aValue).getTime();
			const bDate = new Date(bValue).getTime();
			return sortOrder === 'asc' ? aDate - bDate : bDate - aDate;
		} else {
			const comparison = aValue.localeCompare(bValue, undefined, { numeric: true, sensitivity: 'base' });
			return sortOrder === 'asc' ? comparison : -comparison;
		}
	});
	// Re-append sorted rows
	for (const row of rowsArray) {
		tbody.appendChild(row);
	}
}

inventorySortByInput.addEventListener('change', async () => {
	const sortParams = getInventorySortByParams();
	if (sortParams) {
		sortInventoryTable(sortParams.sortKey, sortParams.sortOrder, sortParams.tbody);
	}
});