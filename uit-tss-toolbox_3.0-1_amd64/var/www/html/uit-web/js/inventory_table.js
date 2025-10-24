async function getInventoryTableData() {
    const response = await fetch('/api/inventory');
    const data = await response.json();
    console.log(data);
}