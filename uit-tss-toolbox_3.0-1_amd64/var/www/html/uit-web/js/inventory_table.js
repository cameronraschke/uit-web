const rowCountElement = document.getElementById('inventory-table-rowcount');
const formAnchor = document.querySelector('#inventory-section');

const urlFilterKeys = [
	"location",
	"department_name",
	"system_manufacturer",
	"system_model",
	"ad_domain",
	"status",
	"is_broken",
	"has_images",
	"csv"
]

async function getInventoryTableData(csvDownload = false, tagnumber, systemSerial, filterLocation, filterDepartment, filterManufacturer, filterModel, filterDomain, filterStatus, filterBroken, filterHasImages) {
	const newURL = new URL(window.location.href);
  const browserQuery = new URLSearchParams();

	if (tagnumber) browserQuery.set("tagnumber", tagnumber);
	if (systemSerial) browserQuery.set("system_serial", systemSerial);
  if (filterLocation) browserQuery.set("location", filterLocation);
  if (filterDepartment) browserQuery.set("department_name", filterDepartment);
  if (filterManufacturer) browserQuery.set("system_manufacturer", filterManufacturer);
  if (filterModel) browserQuery.set("system_model", filterModel);
  if (filterDomain) browserQuery.set("ad_domain", filterDomain);
  if (filterStatus) browserQuery.set("status", filterStatus);
  if (filterBroken) browserQuery.set("is_broken", filterBroken);
  if (filterHasImages) browserQuery.set("has_images", filterHasImages);
  if (csvDownload) browserQuery.set("csv", "true");
	
	const modifiedAPIQuery = new URLSearchParams(browserQuery);
	if (browserQuery.get("update") === "true") {
		modifiedAPIQuery.delete("tagnumber");
		modifiedAPIQuery.delete("system_serial");
	}

  try {
    if (csvDownload) {
      location.href = `/api/inventory?${modifiedAPIQuery.toString()}`;
      return;
    }

		if (browserQuery.toString().length > 0) {
			history.replaceState(null, '', newURL.pathname + '?' + browserQuery.toString());
		} else {
			history.replaceState(null, '', newURL.pathname);
		}

    const response = await fetch(`/api/inventory?${modifiedAPIQuery.toString()}`);
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

		// Row count
    let rowCount = tableDataSorted.length;
    if (!rowCount || rowCount === 0) {
      rowCount = 0;
    }
    rowCountElement.textContent = `${rowCount} entries`;

		// Table body
    tableBody.innerHTML = '';
    const fragment = document.createDocumentFragment();
    for (const jsonRow of tableDataSorted) {
      const tr = document.createElement('tr');
      
      const tagCell = document.createElement('td');
      if (jsonRow.tagnumber) {
				tagCell.dataset.tagnumber = jsonRow.tagnumber;
				const tagCellAnchor = document.createElement('a');
				tagCellAnchor.setAttribute('href', `/inventory?update=true&tagnumber=${encodeURIComponent(jsonRow.tagnumber)}`);
				tagCellAnchor.textContent = jsonRow.tagnumber;
				tagCell.appendChild(tagCellAnchor);
				tagCellAnchor.addEventListener('click', async (event) => {
					event.preventDefault();
					if (event.ctrlKey || event.metaKey) {
						return; // Allow default behavior for Ctrl/Cmd + click
					}
					const tagLookupInput = document.getElementById('inventory-tag-lookup');
					tagLookupInput.value = jsonRow.tagnumber;
					
					try {
						await Promise.all [submitInventoryLookup(), fetchFilteredInventoryData()];
					} catch (error) {
						console.error('Error handling tag click:', error);
					} finally {
						formAnchor.scrollTo(0, 0);
					}
        });
      } else {
        tagCell.dataset.tagnumber = null;
        tagCell.textContent = 'N/A';
      }
      tr.appendChild(tagCell);

      const serialCell = document.createElement('td');
      if (jsonRow.system_serial) {
        serialCell.dataset.system_serial = jsonRow.system_serial;
        if (jsonRow.system_serial.length > 12) {
          serialCell.textContent = jsonRow.system_serial.substring(0, 12) + '...';
          serialCell.title = jsonRow.system_serial;
          serialCell.style.cursor = 'pointer';
          serialCell.addEventListener('click', () => {
            serialCell.textContent = jsonRow.system_serial;
          }, { once: true });
        } else {
          serialCell.textContent = jsonRow.system_serial;
        }
      } else {
        serialCell.dataset.system_serial = null;
        serialCell.textContent = 'N/A';
      }
      tr.appendChild(serialCell);

      const locationCell = document.createElement('td');
      if (jsonRow.location_formatted) {
        locationCell.dataset.location_formatted = jsonRow.location_formatted;
        const locationCellLink = document.createElement('a');
        locationCellLink.setAttribute('href', `/inventory?search=${encodeURIComponent(jsonRow.location_formatted)}`);
        locationCellLink.textContent = jsonRow.location_formatted;
        locationCell.appendChild(locationCellLink);
      } else {
        locationCell.dataset.location_formatted = null;
        locationCell.textContent = 'N/A';
      }
      tr.appendChild(locationCell);

      const manufacturerCell = document.createElement('td');
      if (jsonRow.system_manufacturer && jsonRow.system_model) {
        manufacturerCell.dataset.system_manufacturer = jsonRow.system_manufacturer;
        manufacturerCell.dataset.system_model = jsonRow.system_model;
        if (jsonRow.system_manufacturer.length > 10) {
          const manufacturerText = document.createElement('span');
          manufacturerText.textContent = jsonRow.system_manufacturer.substring(0, 10) + '...';
          manufacturerText.title = jsonRow.system_manufacturer;
          manufacturerText.style.cursor = 'pointer';
          manufacturerCell.appendChild(manufacturerText);
        } else {
          manufacturerCell.textContent = jsonRow.system_manufacturer;
        }

        manufacturerCell.textContent += '/';

        if (jsonRow.system_model.length > 17) {
          const modelText = document.createElement('span');
          modelText.textContent = jsonRow.system_model.substring(0, 17) + '...';
          modelText.title = jsonRow.system_model;
          modelText.style.cursor = 'pointer';
          manufacturerCell.appendChild(modelText);
        } else {
          manufacturerCell.textContent += jsonRow.system_model;
        }

        manufacturerCell.addEventListener('click', () => {
          manufacturerCell.textContent = `${jsonRow.system_manufacturer}/${jsonRow.system_model}`;
        }, { once: true });
      } else {
        manufacturerCell.textContent = 'N/A';
      }
      tr.appendChild(manufacturerCell);

      const departmentCell = document.createElement('td');
      departmentCell.textContent = jsonRow.department_formatted || 'N/A';
      departmentCell.dataset.department_formatted = jsonRow.department_formatted || null;
      tr.appendChild(departmentCell);

      const domainCell = document.createElement('td');
      domainCell.textContent = jsonRow.domain_formatted || 'N/A';
      domainCell.dataset.domain_formatted = jsonRow.domain_formatted || null;
      tr.appendChild(domainCell);

      const statusCell = document.createElement('td');
      statusCell.textContent = jsonRow.status || 'N/A';
      statusCell.dataset.status = jsonRow.status || null;
      tr.appendChild(statusCell);

      const brokenCell = document.createElement('td');
			if (typeof jsonRow.is_broken === 'boolean') {
				brokenCell.textContent = jsonRow.is_broken ? 'Broken' : 'Functional';
				brokenCell.dataset.is_broken = String(jsonRow.is_broken);
			} else {
				brokenCell.textContent = 'N/A';
				brokenCell.dataset.is_broken = 'null';
			}
      tr.appendChild(brokenCell);

      const noteCell = document.createElement('td');
      if (jsonRow.note) {
        noteCell.dataset.note = jsonRow.note;
        if (jsonRow.note.length > 61) {
          noteCell.textContent = jsonRow.note.substring(0, 61) + '...';
          noteCell.title = jsonRow.note;
          noteCell.style.cursor = 'pointer';
        } else {
          noteCell.textContent = jsonRow.note;
        }
        noteCell.addEventListener('click', () => {
          noteCell.textContent = jsonRow.note;
        }, { once: true });
      } else {
        noteCell.dataset.note = null;
        noteCell.textContent = 'N/A';
      }
      tr.appendChild(noteCell);

      const lastUpdateCell = document.createElement('td');
      if (jsonRow.last_updated) {
        const lastUpdateDate = new Date(jsonRow.last_updated);
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