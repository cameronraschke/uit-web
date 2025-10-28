async function getInventoryTableData() {
  try {
    const response = await fetch('/api/inventory');
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

async function renderInventoryTable() {
  const tableBody = document.getElementById('inventory-table-body')
  try {
    const tableData = await getInventoryTableData();
    if (!Array.isArray(tableData) || tableData.length === 0) {
      throw new Error('Table data is empty or invalid.');
    }
    const tableDataSorted = tableData.sort((a, b) => {
      const dateA = new Date(a.last_update || 0);
      const dateB = new Date(b.last_update || 0);
      return dateB - dateA;
    });

    tableBody.innerHTML = '';
    const fragment = document.createDocumentFragment();
    for (const row of tableDataSorted) {
      const tr = document.createElement('tr');
      
      const tagCell = document.createElement('td');
      if (row.tagnumber) {
        tagCell.dataset.tagnumber = row.tagnumber;
        const tagCellLink = document.createElement('a');
        tagCellLink.setAttribute('href', `inventory_form`);
        tagCellLink.textContent = row.tagnumber;
        tagCell.appendChild(tagCellLink);
        tagCell.addEventListener('click', () => {
          const tagLookupInput = document.getElementById('inventory-tag-lookup');
          tagLookupInput.value = row.tagnumber;
          inventoryLookupForm.submit();
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
      if (row.broken === true) {
        brokenCell.textContent = 'Broken';
        brokenCell.dataset.broken = 'true';
      } else if (row.broken === false) {
        brokenCell.textContent = 'Functional';
        brokenCell.dataset.broken = 'false';
      } else {
        brokenCell.textContent = 'N/A';
        brokenCell.dataset.broken = null;
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
    console.error('Error rendering inventory table:', error);
    const errRow = document.createElement('tr');
    const errCell = document.createElement('td');
    errCell.colSpan = 10;
    errCell.textContent = 'No inventory data available: ' + error.message;
    errRow.appendChild(errCell);
    tableBody.replaceChild(errRow, tableBody.firstChild);
    return;
  }
}

Promise.all([renderInventoryTable()]);