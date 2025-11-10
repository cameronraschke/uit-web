const rowCountElement = document.getElementById('inventory-table-rowcount');
const formAnchor = document.querySelector('#inventory-section');

async function getInventoryTableData(csvDownload = false, filterTag, filterSerial, filterLocation, filterDepartment, filterManufacturer, filterModel, filterDomain, filterStatus, filterBroken, filterHasImages) {
  const query = new URLSearchParams();
  if (filterTag) query.append("tagnumber", filterTag);
  if (filterSerial) query.append("system_serial", filterSerial);
  if (filterLocation) query.append("location", filterLocation);
  if (filterDepartment) query.append("department", filterDepartment);
  if (filterManufacturer) query.append("system_manufacturer", filterManufacturer);
  if (filterModel) query.append("system_model", filterModel);
  if (filterDomain) query.append("ad_domain", filterDomain);
  if (filterStatus) query.append("status", filterStatus);
  if (filterBroken) query.append("broken", filterBroken);
  if (filterHasImages) query.append("has_images", filterHasImages);
  if (csvDownload) query.append("csv", "true");
  try {
    if (csvDownload) {
      location.href = `/api/inventory?${query.toString()}`;
      return;
    }
    const response = await fetch(`/api/inventory?${query.toString()}`);
    const rawData = await response.text();
    const jsonData = rawData.trim() ? JSON.parse(rawData) : [];
    if (jsonData && typeof jsonData === 'object' && !Array.isArray(jsonData) && Object.prototype.hasOwnProperty.call(jsonData, 'error')) {
      throw new Error(String(jsonData.error || 'Unknown server error'));
    }
    return jsonData;
  } catch (error) {
    console.error('Error fetching inventory data:', error);
    return [];
  }
}

async function renderInventoryTable(tableData = null) {
  const tableBody = document.getElementById('inventory-table-body')
  try {
    if (!tableData) {
      throw new Error('No table data provided.');
    }
    if (!Array.isArray(tableData) || tableData.length === 0) {
      throw new Error('Table data is empty or invalid.');
    }
    const tableDataSorted = tableData.sort((a, b) => {
      const dateA = new Date(a.last_update || 0);
      const dateB = new Date(b.last_update || 0);
      return dateB - dateA;
    });

    let rowCount = tableDataSorted.length;
    if (!rowCount || rowCount === 0) {
      rowCount = 0;
    }
    rowCountElement.textContent = `${rowCount} entries`;

    tableBody.innerHTML = '';
    const fragment = document.createDocumentFragment();
    for (const row of tableDataSorted) {
      const tr = document.createElement('tr');
      
      const tagCell = document.createElement('td');
      if (row.tagnumber) {
				tagCell.dataset.tagnumber = row.tagnumber;
				const tagCellLink = document.createElement('a');
				tagCellLink.textContent = row.tagnumber;
				tagCell.appendChild(tagCellLink);
				tagCellLink.addEventListener('click', () => {
					const tagLookupInput = document.getElementById('inventory-tag-lookup');
					tagLookupInput.value = row.tagnumber;
					submitInventoryLookup();
    			fetchFilteredInventoryData();
					formAnchor.scrollTo(0, 0);
        });
      } else {
        tagCell.dataset.tagnumber = null;
        tagCell.textContent = 'N/A';
      }
      tr.appendChild(tagCell);

      const serialCell = document.createElement('td');
      if (row.system_serial) {
        serialCell.dataset.system_serial = row.system_serial;
        if (row.system_serial.length > 12) {
          serialCell.textContent = row.system_serial.substring(0, 12) + '...';
          serialCell.title = row.system_serial;
          serialCell.style.cursor = 'pointer';
          serialCell.addEventListener('click', () => {
            serialCell.textContent = row.system_serial;
          }, { once: true });
        } else {
          serialCell.textContent = row.system_serial;
        }
      } else {
        serialCell.dataset.system_serial = null;
        serialCell.textContent = 'N/A';
      }
      tr.appendChild(serialCell);

      const locationCell = document.createElement('td');
      if (row.location_formatted) {
        locationCell.dataset.location_formatted = row.location_formatted;
        const locationCellLink = document.createElement('a');
        locationCellLink.setAttribute('href', `/locations?search=${encodeURIComponent(row.location_formatted)}`);
        locationCellLink.textContent = row.location_formatted;
        locationCell.appendChild(locationCellLink);
      } else {
        locationCell.dataset.location_formatted = null;
        locationCell.textContent = 'N/A';
      }
      tr.appendChild(locationCell);

      const manufacturerCell = document.createElement('td');
      if (row.system_manufacturer && row.system_model) {
        manufacturerCell.dataset.system_manufacturer = row.system_manufacturer;
        manufacturerCell.dataset.system_model = row.system_model;
        if (row.system_manufacturer.length > 10) {
          const manufacturerText = document.createElement('span');
          manufacturerText.textContent = row.system_manufacturer.substring(0, 10) + '...';
          manufacturerText.title = row.system_manufacturer;
          manufacturerText.style.cursor = 'pointer';
          manufacturerCell.appendChild(manufacturerText);
        } else {
          manufacturerCell.textContent = row.system_manufacturer;
        }

        manufacturerCell.textContent += '/';

        if (row.system_model.length > 17) {
          const modelText = document.createElement('span');
          modelText.textContent = row.system_model.substring(0, 17) + '...';
          modelText.title = row.system_model;
          modelText.style.cursor = 'pointer';
          manufacturerCell.appendChild(modelText);
        } else {
          manufacturerCell.textContent += row.system_model;
        }

        manufacturerCell.addEventListener('click', () => {
          manufacturerCell.textContent = `${row.system_manufacturer}/${row.system_model}`;
        }, { once: true });
      } else {
        manufacturerCell.textContent = 'N/A';
      }
      tr.appendChild(manufacturerCell);

      const departmentCell = document.createElement('td');
      departmentCell.textContent = row.department_formatted || 'N/A';
      departmentCell.dataset.department_formatted = row.department_formatted || null;
      tr.appendChild(departmentCell);

      const domainCell = document.createElement('td');
      domainCell.textContent = row.domain_formatted || 'N/A';
      domainCell.dataset.domain_formatted = row.domain_formatted || null;
      tr.appendChild(domainCell);

      const statusCell = document.createElement('td');
      statusCell.textContent = row.status || 'N/A';
      statusCell.dataset.status = row.status || null;
      tr.appendChild(statusCell);

      const brokenCell = document.createElement('td');
			if (typeof row.is_broken === 'boolean') {
				brokenCell.textContent = row.is_broken ? 'Broken' : 'Functional';
				brokenCell.dataset.is_broken = String(row.is_broken);
			} else {
				brokenCell.textContent = 'N/A';
				brokenCell.dataset.is_broken = 'null';
			}
      tr.appendChild(brokenCell);

      const noteCell = document.createElement('td');
      if (row.note) {
        noteCell.dataset.note = row.note;
        if (row.note.length > 61) {
          noteCell.textContent = row.note.substring(0, 61) + '...';
          noteCell.title = row.note;
          noteCell.style.cursor = 'pointer';
        } else {
          noteCell.textContent = row.note;
        }
        noteCell.addEventListener('click', () => {
          noteCell.textContent = row.note;
        }, { once: true });
      } else {
        noteCell.dataset.note = null;
        noteCell.textContent = 'N/A';
      }
      tr.appendChild(noteCell);

      const lastUpdateCell = document.createElement('td');
      if (row.last_updated) {
        const lastUpdateDate = new Date(row.last_updated);
        if (isNaN(lastUpdateDate.getTime())) {
          lastUpdateCell.textContent = 'Unknown date';
        } else {
          const timeFormatted = lastUpdateDate.toLocaleDateString() + ' ' + lastUpdateDate.toLocaleTimeString();
          lastUpdateCell.dataset.last_updated = timeFormatted;
          lastUpdateCell.textContent = timeFormatted;
        }
      } else {
        lastUpdateCell.textContent = 'N/A';
      }
      tr.appendChild(lastUpdateCell);

      fragment.appendChild(tr);
    }
    tableBody.appendChild(fragment);
  } catch (error) {
    // console.error('Error rendering inventory table:', error.message);
    rowCountElement.textContent = `0 entries`;
    tableBody.innerHTML = '';
    const errRow = document.createElement('tr');
    const errCell = document.createElement('td');
    errCell.colSpan = 10;
    errCell.textContent = 'No results found.';
    errRow.appendChild(errCell);
    tableBody.appendChild(errRow);
    return;
  }
}