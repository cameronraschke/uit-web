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
    for (const row of tableDataSorted) {
      const tr = document.createElement('tr');
      
      const tagCell = document.createElement('td');
    }
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