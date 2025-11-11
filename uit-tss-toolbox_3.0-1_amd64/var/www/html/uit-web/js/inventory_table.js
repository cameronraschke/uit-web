const rowCountElement = document.getElementById('inventory-table-rowcount');
const formAnchor = document.querySelector('#inventory-section');

function createTextCell(value, options = {}) {
  const cell = document.createElement('td');
  
  if (!value) {
    cell.textContent = 'N/A';
    cell.dataset[options.datasetKey] = null;
    return cell;
  }
  
  if (options.datasetKey) {
    cell.dataset[options.datasetKey] = value;
  }
  
  if (options.link) {
    const anchor = document.createElement('a');
    anchor.href = options.link;
    anchor.textContent = value;
    if (options.onClick) {
      anchor.addEventListener('click', options.onClick);
    }
    cell.appendChild(anchor);
  } else if (options.truncate && value.length > options.truncate) {
    cell.textContent = value.substring(0, options.truncate) + '...';
    cell.title = value;
    cell.style.cursor = 'pointer';
    cell.addEventListener('click', () => {
      cell.textContent = value;
    }, { once: true });
  } else {
    cell.textContent = value;
  }
  
  return cell;
}

function createManufacturerModelCell(jsonRow) {
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

// Boolean broken status
function createBooleanCell(isBroken) {
  const cell = document.createElement('td');
  
  if (typeof isBroken === 'boolean') {
    cell.textContent = isBroken ? 'Broken' : 'Functional';
    cell.dataset.is_broken = String(isBroken);
  } else {
    cell.textContent = 'N/A';
    cell.dataset.is_broken = 'null';
  }
  
  return cell;
}

// Date formatting
function createTimestampCell(dateValue) {
  const cell = document.createElement('td');
  
  if (!dateValue) {
    cell.textContent = 'N/A';
    return cell;
  }
  
  const date = new Date(dateValue);
  
  if (isNaN(date.getTime())) {
    cell.textContent = 'Unknown date';
  } else {
    const formatted = `${date.toLocaleDateString()} ${date.toLocaleTimeString()}`;
    cell.dataset.last_updated = formatted;
    cell.textContent = formatted;
  }
  
  return cell;
}

// Empty table state
function renderEmptyTable(tableBody, message) {
  rowCountElement.textContent = '0 entries';
  tableBody.innerHTML = '';
  
  const jsonRow = document.createElement('tr');
  const cell = document.createElement('td');
  cell.colSpan = 10;
  cell.textContent = message;
  jsonRow.appendChild(cell);
  tableBody.appendChild(jsonRow);
}


async function renderInventoryTable(tableData = null) {
  const tableBody = document.getElementById('inventory-table-body')
  try {
    if (!tableData) {
      throw new Error('No table data provided.');
    }
    if (!tableData || !Array.isArray(tableData) || tableData.length === 0) {
      throw new Error('Table data is empty or invalid.');
    }
   
		const tableDataSorted = [...tableData].sort((a, b) => 
      new Date(b.last_update || 0) - new Date(a.last_update || 0)
    );

		// Row count
    rowCountElement.textContent = `${tableDataSorted.length} entries`;

		const fragment = document.createDocumentFragment();

		// Table body
    for (const jsonRow of tableDataSorted) {
      const tr = document.createElement('tr');
      
			// Tag Number with link and click handler
      const tagCell = createTextCell(jsonRow.tagnumber, {
        datasetKey: 'tagnumber',
        link: `/inventory?update=true&tagnumber=${encodeURIComponent(jsonRow.tagnumber || '')}`,
        onClick: async (event) => {
          event.preventDefault();
          if (event.ctrlKey || event.metaKey) return;
          
          const tagLookupInput = document.getElementById('inventory-tag-lookup');
          tagLookupInput.value = jsonRow.tagnumber;
          
          try {
            await Promise.all([submitInventoryLookup(), fetchFilteredInventoryData()]);
          } catch (error) {
            console.error('Error handling tag click:', error);
          } finally {
            formAnchor.scrollIntoView({ behavior: 'smooth', block: 'start' });
          }
        }
      });
      tr.appendChild(tagCell);

			// Serial Number with truncation and click to expand
      tr.appendChild(createTextCell(jsonRow.system_serial, {
        datasetKey: 'system_serial',
        truncate: 12
      }));

			// Location with link
      tr.appendChild(createTextCell(jsonRow.location_formatted, {
        datasetKey: 'location_formatted',
        link: `/inventory?location=${encodeURIComponent(jsonRow.location_formatted || '')}`
      }));

			// Manufacturer and Model combined cell
      tr.appendChild(createManufacturerModelCell(jsonRow));

			// Department
      tr.appendChild(createTextCell(jsonRow.department_formatted, {
        datasetKey: 'department_formatted'
      }));

      // Domain
      tr.appendChild(createTextCell(jsonRow.domain_formatted, {
        datasetKey: 'domain_formatted'
      }));

      // Status
      tr.appendChild(createTextCell(jsonRow.status, {
        datasetKey: 'status'
      }));

      // Broken status
      tr.appendChild(createBooleanCell(jsonRow.is_broken));

      // Note (truncated)
      tr.appendChild(createTextCell(jsonRow.note, {
        datasetKey: 'note',
        truncate: 61
      }));

      // Last Updated
      tr.appendChild(createTimestampCell(jsonRow.last_updated));

      fragment.appendChild(tr);
    }
    tableBody.innerHTML = '';
    tableBody.appendChild(fragment);
  } catch (error) {
		console.error('Error rendering inventory table:', error);
    renderEmptyTable(tableBody, 'No results found.');
  }
}