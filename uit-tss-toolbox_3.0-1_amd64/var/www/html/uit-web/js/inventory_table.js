async function getInventoryTableData(filterTag, filterSerial, filterLocation, filterDepartment, filterManufacturer, filterModel, filterDomain, filterStatus, filterBroken, filterHasImages) {
  const query = new URLSearchParams();
  if (filterTag) query.append("tagnumber", filterTag);
  if (filterSerial) query.append("system_serial", filterSerial);
  if (filterLocation) query.append("location", filterLocation);
  if (filterDepartment) query.append("department", filterDepartment);
  if (filterManufacturer) query.append("system_manufacturer", filterManufacturer);
  if (filterModel) query.append("system_model", filterModel);
  if (filterDomain) query.append("domain", filterDomain);
  if (filterStatus) query.append("status", filterStatus);
  if (filterBroken) query.append("broken", filterBroken);
  if (filterHasImages) query.append("has_images", filterHasImages);
  try {
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

    tableBody.innerHTML = '';
    const fragment = document.createDocumentFragment();
    for (const row of tableDataSorted) {
      const tr = document.createElement('tr');
      
      const tagCell = document.createElement('td');
      if (row.tagnumber) {
        tagCell.dataset.tagnumber = row.tagnumber;
        const tagCellLink = document.createElement('a');
        tagCellLink.setAttribute('href', `#inventory-section`);
        tagCellLink.textContent = row.tagnumber;
        tagCell.appendChild(tagCellLink);
        tagCell.addEventListener('click', () => {
          const tagLookupInput = document.getElementById('inventory-tag-lookup');
          tagLookupInput.value = row.tagnumber;
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
    // console.error('Error rendering inventory table:', error.message);
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

async function fetchFilteredInventoryData() {
  const tag = filterTag.value.trim() || null;
  const serial = filterSerial.value.trim() || null;
  const location = filterLocation.value.trim() || null;
  const department = filterDepartment.value.trim() || null;
  const manufacturer = filterManufacturer.value.trim() || null;
  const model = filterModel.value.trim() || null;
  const domain = filterDomain.value.trim() || null;
  const status = filterStatus.value.trim() || null;
  const broken = filterBroken.value.trim() || null;
  const hasImages = filterHasImages.value.trim() || null;

  try {
    const tableData = await getInventoryTableData(tag, serial, location, department, manufacturer, model, domain, status, broken, hasImages);
    await renderInventoryTable(tableData);
  } catch (error) {
    console.error("Error fetching inventory data:", error);
  }
}

const inventoryFilterForm = document.getElementById('inventory-filter-form');
inventoryFilterForm.addEventListener("submit", async (event) => {
  event.preventDefault();
  await fetchFilteredInventoryData();
});

const inventoryFilterResetButton = document.getElementById('inventory-filter-form-reset-button');
inventoryFilterResetButton.addEventListener("click", async (event) => {
  event.preventDefault();
  inventoryFilterForm.reset();
  document.querySelectorAll('.inventory-filter-reset').forEach(elem => {
    elem.style.display = 'none';
  });
  await fetchFilteredInventoryData();
});

function populateManufacturerSelect() {
  const manufacturerSelect = document.getElementById('inventory-filter-manufacturer');
  if (!manufacturerSelect) return;

  // Get manufacturers
  const manufacturersMap = new Map();
  allModelsData.forEach(item => {
    if (item.system_manufacturer && !manufacturersMap.has(item.system_manufacturer)) {
      manufacturersMap.set(item.system_manufacturer, item.system_manufacturer_formatted || item.system_manufacturer);
    }
  });

  // Clear and rebuild manufacturer select options
  manufacturerSelect.innerHTML = '';
  const defaultOption = document.createElement('option');
  defaultOption.value = '';
  defaultOption.textContent = 'Manufacturer';
  defaultOption.selected = true;
  defaultOption.disabled = true;
  manufacturerSelect.appendChild(defaultOption);

  // Sort by formatted name
  const sortedManufacturers = Array.from(manufacturersMap.entries()).sort((a, b) => 
    a[1].localeCompare(b[1])
  );

  sortedManufacturers.forEach(([manufacturer, manufacturerFormatted]) => {
    const option = document.createElement('option');
    option.value = manufacturer;
    option.textContent = manufacturerFormatted;
    manufacturerSelect.appendChild(option);
  });
}

function populateModelSelect(selectedManufacturer = null) {
  const modelSelect = document.getElementById('inventory-filter-model');
  if (!modelSelect) return;

  // Filter models by manufacturer if one is selected
  const filteredModels = selectedManufacturer
    ? allModelsData.filter(item => item.system_manufacturer === selectedManufacturer)
    : allModelsData;

  // Get models
  const modelsMap = new Map();
  filteredModels.forEach(item => {
    if (item.system_model && !modelsMap.has(item.system_model)) {
      modelsMap.set(item.system_model, item.system_model_formatted || item.system_model);
    }
  });

  // Clear and rebuild model select options
  modelSelect.innerHTML = '';
  const defaultOption = document.createElement('option');
  defaultOption.value = '';
  defaultOption.textContent = 'Model';
  defaultOption.selected = true;
  defaultOption.disabled = true;
  modelSelect.appendChild(defaultOption);

  // Sort by formatted name
  const sortedModels = Array.from(modelsMap.entries()).sort((a, b) => 
    a[1].localeCompare(b[1])
  );

  sortedModels.forEach(([model, modelFormatted]) => {
    const option = document.createElement('option');
    option.value = model;
    option.textContent = modelFormatted;
    modelSelect.appendChild(option);
  });
}


async function loadManufacturersAndModels() {
  try {
    const response = await fetchData('/api/models');
    if (!response) {
      throw new Error('No data returned from /api/models');
    }

    allModelsData = Array.isArray(response) ? response : [];
    populateManufacturerSelect();
    populateModelSelect();

  } catch (error) {
    console.error('Error fetching models:', error);
  }
}

async function updateModelOptionsBasedOnManufacturer() {
  const manufacturerSelect = document.getElementById('inventory-filter-manufacturer');
  const selectedManufacturer = manufacturerSelect.value || null;
  
  // Repopulate model select with filtered options
  populateModelSelect(selectedManufacturer);
}

filterManufacturer.addEventListener("change", async (event) => {
  await updateModelOptionsBasedOnManufacturer();
});
