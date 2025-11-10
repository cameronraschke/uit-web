let allModelsData = [];

const currentURL = new URL(window.location.href);
const queryParams = new URLSearchParams(currentURL.search);

function updateURLParameters(urlParameter, value) {
	const newURL = new URL(window.location.href);
	if (value) {
		newURL.searchParams.set(urlParameter, value);
	} else {
		newURL.searchParams.delete(urlParameter);
	}
	if (newURL.searchParams.toString()) {
		history.replaceState(null, '', newURL.pathname + '?' + newURL.searchParams.toString());
	} else {
		history.replaceState(null, '', newURL.pathname);
	}
	
}

// Location filter
const filterLocation = document.getElementById('inventory-filter-location')
filterLocation.value = queryParams.get('location_formatted') || '';
const filterLocationReset = document.getElementById('inventory-filter-location-reset')
createFilterResetHandler(filterLocation, filterLocationReset);

// Department filter
const filterDepartment = document.getElementById('inventory-filter-department')
filterDepartment.value = queryParams.get('department') || '';
const filterDepartmentReset = document.getElementById('inventory-filter-department-reset')
createFilterResetHandler(filterDepartment, filterDepartmentReset);

// Manufacturer & model filters
const filterManufacturer = document.getElementById('inventory-filter-manufacturer')
filterManufacturer.value = queryParams.get('system_manufacturer') || '';
const filterManufacturerReset = document.getElementById('inventory-filter-manufacturer-reset')
const filterModel = document.getElementById('inventory-filter-model')
filterModel.value = queryParams.get('system_model') || '';
const filterModelReset = document.getElementById('inventory-filter-model-reset')
createFilterResetHandler(filterManufacturer, filterManufacturerReset);
createFilterResetHandler(filterModel, filterModelReset);

// Domain filter
const filterDomain = document.getElementById('inventory-filter-domain')
filterDomain.value = queryParams.get('ad_domain') || '';
const filterDomainReset = document.getElementById('inventory-filter-domain-reset')
createFilterResetHandler(filterDomain, filterDomainReset);

// Status filter
const filterStatus = document.getElementById('inventory-filter-status')
filterStatus.value = queryParams.get('status') || '';
const filterStatusReset = document.getElementById('inventory-filter-status-reset')
createFilterResetHandler(filterStatus, filterStatusReset);

// Broken filter
const filterBroken = document.getElementById('inventory-filter-broken')
filterBroken.value = queryParams.get('is_broken') || '';
const filterBrokenReset = document.getElementById('inventory-filter-broken-reset')
createFilterResetHandler(filterBroken, filterBrokenReset);

// Has Images filter
const filterHasImages = document.getElementById('inventory-filter-has_images')
filterHasImages.value = queryParams.get('has_images') || '';
const filterHasImagesReset = document.getElementById('inventory-filter-has_images-reset')
createFilterResetHandler(filterHasImages, filterHasImagesReset);

// Reset filter
function createFilterResetHandler(filterInput, resetButton) {
  filterInput.addEventListener("change", async () => {
    resetButton.style.display = 'inline-block';
		if (filterInput === filterManufacturer) {
			populateModelSelect(filterInput.value || null);
		}
    await fetchFilteredInventoryData();
  });
  
  resetButton.addEventListener("click", async (event) => {
    event.preventDefault();
    resetButton.style.display = 'none';
		filterInput.value = '';
    await fetchFilteredInventoryData();
  });
}

async function fetchFilteredInventoryData(csvDownload = false) {
  const location = filterLocation.value.trim() || null;
	updateURLParameters('location', location);
  const department = filterDepartment.value.trim() || null;
	updateURLParameters('department_name', department);
  const manufacturer = filterManufacturer.value.trim() || null;
	updateURLParameters('system_manufacturer', manufacturer);
  const model = filterModel.value.trim() || null;
	updateURLParameters('system_model', model);
  const domain = filterDomain.value.trim() || null;
	updateURLParameters('ad_domain', domain);
  const status = filterStatus.value.trim() || null;
	updateURLParameters('status', status);
  const broken = filterBroken.value.trim() || null;
	updateURLParameters('is_broken', broken);
  const hasImages = filterHasImages.value.trim() || null;
	updateURLParameters('has_images', hasImages);

  try {
		const tableData = await getInventoryTableData(
      csvDownload,
      location,
      department,
      manufacturer,
      model,
      domain,
      status,
      broken,
      hasImages
    );
    await renderInventoryTable(tableData);
  } catch (error) {
    console.error("Error fetching inventory data:", error);
  }
}

const inventoryFilterForm = document.getElementById('inventory-filter-form');
inventoryFilterForm.addEventListener("submit", async (event) => {
  event.preventDefault();
  fetchFilteredInventoryData();
});

const inventoryFilterResetButton = document.getElementById('inventory-filter-form-reset-button');
inventoryFilterResetButton.addEventListener("click", async (event) => {
  event.preventDefault();
	history.replaceState(null, '', window.location.pathname);
  inventoryFilterForm.reset();
  document.querySelectorAll('.inventory-filter-reset').forEach(elem => {
    elem.style.display = 'none';
  });
	await Promise.all([
  	fetchFilteredInventoryData()
	]);
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