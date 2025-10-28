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
        tagCellLink.setAttribute('href', `/client/${encodeURIComponent(row.tagnumber)}`);
        tagCellLink.textContent = row.tagnumber;
        tagCell.appendChild(tagCellLink);
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
        if (row.system_manufacturer.length > 9) {
          const manufacturerText = document.createElement('span');
          manufacturerText.textContent = row.system_manufacturer.substring(0, 9) + '...';
          manufacturerText.title = row.system_manufacturer;
          manufacturerCell.appendChild(manufacturerText);
        } else {
          manufacturerCell.textContent = row.system_manufacturer;
        }

        manufacturerCell.textContent += '/';

        if (row.system_model.length > 15) {
          const modelText = document.createElement('span');
          modelText.textContent = row.system_model.substring(0, 15) + '...';
          modelText.title = row.system_model;
          manufacturerCell.appendChild(modelText);
        } else {
          manufacturerCell.textContent += row.system_model;
        }
      } else {
        manufacturerCell.textContent = 'N/A';
      }
      tr.appendChild(manufacturerCell);

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