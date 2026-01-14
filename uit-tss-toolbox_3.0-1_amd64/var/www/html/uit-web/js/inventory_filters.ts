type Domain = {
	domain_name: string;
	domain_name_formatted: string;
	domain_sort_order: number;
};

type ManufacturersAndModels = {
	system_model: string;
	system_model_formatted: string;
	system_manufacturer: string;
	system_manufacturer_formatted: string;
};

type ManufacturerAndModelsCache = {
	timestamp: number;
	manufacturers_and_models: ManufacturersAndModels[];
};

const inventoryFilterForm = document.getElementById('inventory-search-form') as HTMLFormElement;
const filterLocation = document.getElementById('inventory-search-location') as HTMLSelectElement;
const filterLocationReset = document.getElementById('inventory-search-location-reset') as HTMLElement;
const filterDepartment = document.getElementById('inventory-search-department') as HTMLSelectElement;
const filterDepartmentReset = document.getElementById('inventory-search-department-reset') as HTMLElement;
const filterManufacturer = document.getElementById('inventory-search-manufacturer') as HTMLSelectElement;
const filterManufacturerReset = document.getElementById('inventory-search-manufacturer-reset') as HTMLElement;
const filterModel = document.getElementById('inventory-search-model') as HTMLSelectElement;
const filterModelReset = document.getElementById('inventory-search-model-reset') as HTMLElement;
const filterDomain = document.getElementById('inventory-search-domain') as HTMLSelectElement;
const filterDomainReset = document.getElementById('inventory-search-domain-reset') as HTMLElement;
const filterStatus = document.getElementById('inventory-search-status') as HTMLSelectElement;
const filterStatusReset = document.getElementById('inventory-search-status-reset') as HTMLElement;
const filterBroken = document.getElementById('inventory-search-broken') as HTMLSelectElement;
const filterBrokenReset = document.getElementById('inventory-search-broken-reset') as HTMLElement;
const filterHasImages = document.getElementById('inventory-search-has_images') as HTMLSelectElement;
const filterHasImagesReset = document.getElementById('inventory-search-has_images-reset') as HTMLElement;

let allModelsData: string[] = [];

const currentURL = new URL(window.location.href);
const queryParams = new URLSearchParams(currentURL.search);

function updateURLParameters(urlParameter: string | null, value: string | null) {
	const newURL = new URL(window.location.href);
	if (urlParameter && value) {
		newURL.searchParams.set(urlParameter, value);
	} else if (urlParameter && !value) {
		newURL.searchParams.delete(urlParameter);
	}
	if (newURL.searchParams.toString()) {
		history.pushState(null, '', newURL.pathname + '?' + newURL.searchParams.toString());
	} else {
		history.replaceState(null, '', newURL.pathname);
	}
}

function initializeSearch() {
	setFiltersFromURL();

	createFilterResetHandler(filterLocation, filterLocationReset);
	createFilterResetHandler(filterDepartment, filterDepartmentReset);
	createFilterResetHandler(filterManufacturer, filterManufacturerReset);
	createFilterResetHandler(filterModel, filterModelReset);
	createFilterResetHandler(filterDomain, filterDomainReset);
	createFilterResetHandler(filterStatus, filterStatusReset);
	createFilterResetHandler(filterBroken, filterBrokenReset);
	createFilterResetHandler(filterHasImages, filterHasImagesReset);

	filterModel.disabled = !filterManufacturer.value;
}

function createFilterResetHandler(filterElement: HTMLSelectElement, resetButton: HTMLElement) {
	if (!filterElement || !resetButton) {
		console.error("Filter element or reset button not found.");
		return;
	}

	if (filterElement.value && filterElement.value.length > 0) {
		resetButton.style.display = 'inline-block';
	}

	filterElement.addEventListener("change", () => {
		resetButton.style.display = 'inline-block';
		const paramName = getURLParamName(filterElement);
		updateURLParameters(paramName, filterElement.value);
		if (filterElement === filterManufacturerReset || filterElement == filterModelReset) {
			populateManufacturerSelect().catch((error) => {
				console.error("Error populating manufacturer select:", error);
			});
			populateModelSelect().catch((error) => {
				console.error("Error populating model select:", error);
			});
			return;
		}
		fetchFilteredInventoryData().catch((error) => {
			console.error("Error fetching filtered inventory data:", error);
		});
	});
  
	resetButton.addEventListener("click", (event) => {
		event.preventDefault();
		resetButton.style.display = 'none';
		filterElement.value = '';
		const paramName = getURLParamName(filterElement);
		updateURLParameters(paramName, null);
		if (resetButton === filterManufacturerReset || resetButton == filterModelReset) {
			populateManufacturerSelect().catch((error) => {
				console.error("Error populating manufacturer select:", error);
			});
			populateModelSelect().catch((error) => {
				console.error("Error populating model select:", error);
			});
			return;
		}
		fetchFilteredInventoryData().catch((error) => {
			console.error("Error fetching filtered inventory data:", error);
		});
	});
}

function getURLParamName(filterElement: HTMLSelectElement): string {
	if (filterElement === filterLocation) return 'location';
	if (filterElement === filterDepartment) return 'department_name';
	if (filterElement === filterManufacturer) return 'system_manufacturer';
	if (filterElement === filterModel) return 'system_model';
	if (filterElement === filterDomain) return 'ad_domain';
	if (filterElement === filterStatus) return 'status';
	if (filterElement === filterBroken) return 'is_broken';
	if (filterElement === filterHasImages) return 'has_images';
	return '';
}

function setFiltersFromURL(): void {
	const currentParams = new URLSearchParams(window.location.search);
	filterLocation.value = currentParams.get('location') || '';
	filterDepartment.value = currentParams.get('department_name') || '';
	filterManufacturer.value = currentParams.get('system_manufacturer') || '';
	filterModel.value = currentParams.get('system_model') || '';
	filterDomain.value = currentParams.get('ad_domain') || '';
	filterStatus.value = currentParams.get('status') || '';
	filterBroken.value = currentParams.get('is_broken') || '';
	filterHasImages.value = currentParams.get('has_images') || '';
}

async function fetchFilteredInventoryData(csvDownload = false): Promise<void> {
	const currentParams = new URLSearchParams(window.location.search);

	const update = currentParams.get('update');
	const tagnumber = currentParams.get('tagnumber') || null;
	const systemSerial = currentParams.get('system_serial') || null;

	const location = filterLocation.value.trim() || null;
	const department = filterDepartment.value.trim() || null;
	const manufacturer = filterManufacturer.value.trim() || null;
	const model = filterModel.value.trim() || null;
	const domain = filterDomain.value.trim() || null;
	const status = filterStatus.value.trim() || null;
	const broken = filterBroken.value.trim() || null;
	const hasImages = filterHasImages.value.trim() || null;

	const browserQuery = new URLSearchParams();
	if (update) browserQuery.set('update', update);
	if (tagnumber) browserQuery.set('tagnumber', tagnumber);
	if (systemSerial) browserQuery.set('system_serial', systemSerial);
	if (location) browserQuery.set('location', location);
	if (department) browserQuery.set('department_name', department);
	if (manufacturer) browserQuery.set('system_manufacturer', manufacturer);
	if (model) browserQuery.set('system_model', model);
	if (domain) browserQuery.set('ad_domain', domain);
	if (status) browserQuery.set('status', status);
	if (broken) browserQuery.set('is_broken', broken);
	if (hasImages) browserQuery.set('has_images', hasImages);

	const apiQuery = new URLSearchParams(browserQuery); // Copy
	if (update === "true") {
		apiQuery.delete("update");
		apiQuery.delete("tagnumber");
		apiQuery.delete("system_serial");
	}

	const newURL = new URL(window.location.href);
	newURL.search = browserQuery.toString();
	if (browserQuery.toString()) {
		history.replaceState(null, '', newURL.pathname + '?' + browserQuery.toString());
	} else {
		history.replaceState(null, '', newURL.pathname);
	}

	if (csvDownload) {
		window.location.href = `/api/inventory?csv=true&${apiQuery.toString()}`;
		return;
	}

  try {
    const response = await fetch(`/api/inventory?${apiQuery.toString()}`);
    const rawData = await response.text();
    const jsonData = rawData.trim() ? JSON.parse(rawData) : [];
    if (jsonData && typeof jsonData === 'object' && !Array.isArray(jsonData) && Object.prototype.hasOwnProperty.call(jsonData, 'error')) {
      throw new Error(String(jsonData.error || 'Unknown server error'));
    }
    await renderInventoryTable(jsonData);
  } catch (error) {
    console.error("Error fetching inventory data:", error);
  }
}

inventoryFilterForm.addEventListener("submit", (event) => {
  event.preventDefault();
  fetchFilteredInventoryData();
});

const inventoryFilterResetButton = document.getElementById('inventory-search-form-reset-button') as HTMLElement;
inventoryFilterResetButton.addEventListener("click", async (event) => {
  event.preventDefault();
	history.replaceState(null, '', window.location.pathname);
  inventoryFilterForm.reset();
  document.querySelectorAll('.inventory-search-reset').forEach((elem: HTMLElement) => {
    elem.style.display = 'none';
  });
	currentURL.search = '';
	await populateManufacturerSelect();
	await populateModelSelect();

  filterModel.innerHTML = '';
  const defaultOption = document.createElement('option');
  defaultOption.value = '';
  defaultOption.textContent = 'Model';
  defaultOption.selected = true;
  filterModel.appendChild(defaultOption);
	filterModel.disabled = true;
	
  await fetchFilteredInventoryData();
});

async function fetchAllManufacturersAndModels(): Promise<Array<ManufacturersAndModels> | []> {
	const cached = sessionStorage.getItem("uit_manufacturers_and_models");

  try {
		if (cached) {
			const cacheEntry: ManufacturerAndModelsCache = JSON.parse(cached);
			if (Date.now() - cacheEntry.timestamp < 300000 && Array.isArray(cacheEntry.manufacturers_and_models)) {
				console.log("Loaded manufacturers and models from cache");
				return cacheEntry.manufacturers_and_models;
			}
		}

    const data: ManufacturersAndModels[] = await fetchData('/api/models');
    if (!data || !Array.isArray(data) || data.length === 0) {
      throw new Error('No data returned from /api/models');
    }
		const cacheEntry: ManufacturerAndModelsCache = {
			timestamp: Date.now(),
			manufacturers_and_models: data
		};
		sessionStorage.setItem("uit_manufacturers_and_models", JSON.stringify(cacheEntry));
		console.log("Cached manufacturers and models data");
    return data;
  } catch (error) {
    console.error('Error fetching manufacturers and models:', error);
		return [];
  }
}

async function populateManufacturerSelect() {
  if (!filterManufacturer) return;

	const initialValue = filterManufacturer.value;

	filterManufacturer.disabled = true;

  const data: ManufacturersAndModels[] = await fetchAllManufacturersAndModels();
	if (!data || !Array.isArray(data) || data.length === 0) return;

	// Sort manufacturers array
	data.sort((a, b) => {
		const manufacturerA = a.system_manufacturer_formatted || a.system_manufacturer;
		const manufacturerB = b.system_manufacturer_formatted || b.system_manufacturer;
		return manufacturerA.localeCompare(manufacturerB);
	});

  // Clear and rebuild manufacturer select options
  resetSelectElement(filterManufacturer, 'Manufacturer');

  // Sort by formatted name
  for (const item of data) {
		if (!item.system_manufacturer || !item.system_manufacturer_formatted) continue;
		const option = document.createElement('option');
		option.value = item.system_manufacturer;
		option.textContent = item.system_manufacturer_formatted || item.system_manufacturer;
		filterManufacturer.appendChild(option);
	}

  filterManufacturer.value = (initialValue && data.some(item => item.system_manufacturer === initialValue)) ? initialValue : '';
	if (filterManufacturer.value !== '') {
		updateURLParameters('system_manufacturer', filterManufacturer.value);
	} else {
		updateURLParameters('system_manufacturer', null);
	}
	filterManufacturer.disabled = false;
}



async function populateModelSelect() {
  if (!filterModel) return;
	
	const initialValue = filterModel.value;

	if (!filterManufacturer.value || filterManufacturer.value.trim().length === 0) {
		resetSelectElement(filterModel, 'Model', true);
		updateURLParameters('system_model', null);
		return;
	}

	filterModel.disabled = true;

  const data: ManufacturersAndModels[] = await fetchAllManufacturersAndModels();
	if (!data || !Array.isArray(data) || data.length === 0) return;

	data.sort((a, b) => {
		const modelA = a.system_model_formatted || a.system_model;
		const modelB = b.system_model_formatted || b.system_model;
		return modelA.localeCompare(modelB);
	});

	resetSelectElement(filterModel, 'Model');

	for (const item of data) {
		if (item.system_manufacturer !== filterManufacturer.value) continue;
		if (!item.system_model || !item.system_model_formatted) continue;
		const option = document.createElement('option');
		option.value = item.system_model;
		option.textContent = item.system_model_formatted || item.system_model;
		filterModel.appendChild(option);
	}

	filterModel.value = (initialValue && data.some(item => item.system_model === initialValue)) ? initialValue : '';
	filterModel.disabled = false;
}


async function populateDomainSelect(elem: HTMLSelectElement) {
	if (!elem) return;
	try {
		const response = await fetchData('/api/domains');
		if (!response) {
			throw new Error('No data returned from /api/domains');
		}
		const domainsData: Domain[] = Array.isArray(response) ? response : [];
		elem.innerHTML = '';
		const defaultOption = document.createElement('option');
		defaultOption.value = '';
		defaultOption.textContent = 'AD Domain';
		defaultOption.selected = true;
		elem.addEventListener('click', () => {
			defaultOption.disabled = true;
		});
		elem.appendChild(defaultOption);
		domainsData.sort((a, b) => a.domain_sort_order - b.domain_sort_order);
		domainsData.forEach((domain) => {
			const option = document.createElement('option');
			option.value = domain.domain_name;
			option.textContent = domain.domain_name_formatted || domain.domain_name;
			elem.appendChild(option);
		});
	} catch (error) {
		console.error('Error fetching domains:', error);
	}
}