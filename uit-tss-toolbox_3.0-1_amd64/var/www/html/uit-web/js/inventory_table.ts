type InventoryRow = {
	tagnumber: number | 0;
	system_serial: string | "";
	location_formatted: string | "";
	system_manufacturer: string | "";
	system_model: string | "";
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

			// Tag Number URL with system serial as well
			const tagCell = document.createElement('td');
			tagCell.dataset.tagnumber = jsonRow.tagnumber ? jsonRow.tagnumber.toString() : 'No Tag';
			tagCell.appendChild(document.createElement('a'));
			const tagAnchor = tagCell.querySelector('a') as HTMLAnchorElement;
			tagAnchor.textContent = jsonRow.tagnumber ? jsonRow.tagnumber.toString() : 'No Tag';
			const tagURL = new URL(window.location.href);
			tagURL.searchParams.set('tagnumber', jsonRow.tagnumber.toString() || '');
			tagURL.searchParams.set('system_serial', jsonRow.system_serial || '');
			tagURL.searchParams.set('update', 'true');
			tagAnchor.href = tagURL.toString();

			tr.appendChild(tagCell);

			// Serial Number with truncation and click to expand
			tr.appendChild(createTextCell(undefined, 'system_serial', jsonRow.system_serial, 20, undefined));

			// Location
			tr.appendChild(createTextCell(undefined, 'location', jsonRow.location_formatted, 40, undefined));

			// Manufacturer and Model combined cell
			const manufacturerModelCell = document.createElement('td');
			manufacturerModelCell.dataset.systemManufacturer = jsonRow.system_manufacturer || '';
			manufacturerModelCell.dataset.systemModel = jsonRow.system_model || '';
			let manufacturerModelText = 'N/A';
			if (jsonRow.system_manufacturer && jsonRow.system_model) {
				manufacturerModelText = `${jsonRow.system_manufacturer}/${jsonRow.system_model}`;
			} else if (jsonRow.system_manufacturer && !jsonRow.system_model) {
				manufacturerModelText = `${jsonRow.system_manufacturer}/Unknown Model`;
			} else if (!jsonRow.system_manufacturer && jsonRow.system_model) {
				manufacturerModelText = `Unknown Manufacturer/${jsonRow.system_model}`;
			} else {
				manufacturerModelCell.style.fontStyle = 'italic';
			}
			if (manufacturerModelText.length > 30) {
				const arr = manufacturerModelText.split('/');
				const truncated = `${arr[0].substring(0, 11)}.../${arr[1].substring(0, 17)}...`;
				manufacturerModelCell.title = manufacturerModelText;
				manufacturerModelCell.style.cursor = 'pointer';
				manufacturerModelCell.textContent = truncated;
				manufacturerModelCell.addEventListener('click', () => {
					manufacturerModelCell.textContent = manufacturerModelText;
					manufacturerModelCell.style.cursor = 'auto';
				}, { once: true });
			} else {
				manufacturerModelCell.textContent = manufacturerModelText;
			}
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