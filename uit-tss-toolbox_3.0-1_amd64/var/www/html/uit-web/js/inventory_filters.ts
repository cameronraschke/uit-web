type Domain = {
	ad_domain: string;
	ad_domain_formatted: string;
	domain_sort_order: number;
};

type DomainCache = {
	timestamp: number;
	domains: Domain[];
};

type ManufacturersAndModels = {
	system_manufacturer: string;
	system_model: string;
	system_model_count: number;
	system_manufacturer_count?: number;
};

type ManufacturerAndModelsCache = {
	timestamp: number;
	manufacturers_and_models: ManufacturersAndModels[];
};

type Status = {
	status: string;
	status_formatted: string;
	sort_order: number;
};

type StatusCache = {
	timestamp: number;
	statuses: Status[];
}

type AdvSearchFilterParams = {
	inputElement: HTMLSelectElement;
	resetElement: HTMLElement;
	paramString: string;
};

const inventoryFilterForm = document.getElementById('adv-search-form') as HTMLFormElement;
const advSearchFormReset = document.getElementById('adv-search-form-reset-button') as HTMLElement;
const advSearchLocation = document.getElementById('adv-search-location') as HTMLSelectElement;
const advSearchLocationReset = document.getElementById('adv-search-location-reset') as HTMLElement;
const filterDepartment = document.getElementById('adv-search-department') as HTMLSelectElement;
const filterDepartmentReset = document.getElementById('adv-search-department-reset') as HTMLElement;
const filterManufacturer = document.getElementById('adv-search-manufacturer') as HTMLSelectElement;
const filterManufacturerReset = document.getElementById('adv-search-manufacturer-reset') as HTMLElement;
const filterModel = document.getElementById('adv-search-model') as HTMLSelectElement;
const filterModelReset = document.getElementById('adv-search-model-reset') as HTMLElement;
const filterDomain = document.getElementById('adv-search-ad-domain') as HTMLSelectElement;
const filterDomainReset = document.getElementById('adv-search-ad-domain-reset') as HTMLElement;
const filterStatus = document.getElementById('adv-search-status') as HTMLSelectElement;
const filterStatusReset = document.getElementById('adv-search-status-reset') as HTMLElement;
const filterBroken = document.getElementById('adv-search-is-broken') as HTMLSelectElement;
const filterBrokenReset = document.getElementById('adv-search-is-broken-reset') as HTMLElement;
const filterHasImages = document.getElementById('adv-search-has-images') as HTMLSelectElement;
const filterHasImagesReset = document.getElementById('adv-search-has-images-reset') as HTMLElement;

const advSearchParams: AdvSearchFilterParams[] = [
	{ inputElement: advSearchLocation, resetElement: advSearchLocationReset, paramString: 'location' },
	{ inputElement: filterDepartment, resetElement: filterDepartmentReset, paramString: 'department_name' },
	{ inputElement: filterManufacturer, resetElement: filterManufacturerReset, paramString: 'system_manufacturer' },
	{ inputElement: filterModel, resetElement: filterModelReset, paramString: 'system_model' },
	{ inputElement: filterDomain, resetElement: filterDomainReset, paramString: 'ad_domain' },
	{ inputElement: filterStatus, resetElement: filterStatusReset, paramString: 'status' },
	{ inputElement: filterBroken, resetElement: filterBrokenReset, paramString: 'is_broken' },
	{ inputElement: filterHasImages, resetElement: filterHasImagesReset, paramString: 'has_images' }
];

let allModelsData: string[] = [];

function resetAdvSearchURLParameters() {
	for (const param of advSearchParams) {
		if (!param.paramString) continue;
		setURLParameter(param.paramString, null);
	}
}

function updateFiltersFromURL() {
	const currentParams = new URLSearchParams(window.location.search);
	for (const param of advSearchParams) {
		if (!param.inputElement || !param.paramString) continue;
		const urlValue = currentParams.get(param.paramString);
		if (urlValue && urlValue.trim().length > 0) param.inputElement.value = urlValue;
		handleAdvSearchInputChange(param.inputElement, param.resetElement);
	}
}

function handleAdvSearchInputChange(filterEl: HTMLSelectElement, resetEl: HTMLElement) {
	if (!filterEl || !resetEl) {
		console.warn("Filter inputElement or reset button not found.");
		return;
	}
	const paramString = getURLParamName(filterEl);
	// Testing a string here, otherwise "false" would not show the reset button
	if (filterEl.value !== '' && filterEl.value.trim().length > 0) {
		setURLParameter(paramString, filterEl.value);
		resetEl.style.display = 'inline-block';
		filterEl.classList.add('changed-input');
	} else {
		setURLParameter(paramString, null);
		resetEl.style.display = 'none';
		filterEl.classList.remove('changed-input');
	}
}

function initializeAdvSearchListeners(filterEl: HTMLSelectElement, resetEl: HTMLElement) {
	filterEl.addEventListener("change", async () => {
		handleAdvSearchInputChange(filterEl, resetEl);
		try {
			if (filterEl === filterManufacturer || filterEl === filterModel) {
				await Promise.all([populateManufacturerSelect(filterManufacturer, filterManufacturerReset).then(() => populateModelSelect(filterModel, filterModelReset)), renderInventoryTable()]);
			} else {
				await renderInventoryTable();
			}
		} catch (err) {
			console.error(`Error fetching data from filterEl on change event listener:`, err);
		}
	});
  
	resetEl.addEventListener("click", async (event) => {
		event.preventDefault();
		filterEl.value = "";
		handleAdvSearchInputChange(filterEl, resetEl);
		try {
			if (filterEl === filterManufacturer || filterEl === filterModel) {
				await Promise.all([populateManufacturerSelect(filterManufacturer, filterManufacturerReset).then(() => populateModelSelect(filterModel, filterModelReset)), renderInventoryTable()]);
			} else {
				await renderInventoryTable();
			}
		} catch (err) {
			console.error(`Error fetching data from filterEl on change event listener:`, err);
		}
	});
}

async function fetchFilteredInventoryData(csvDownload = false): Promise<InventoryRow[] | null> {
	const currentParams = new URLSearchParams(window.location.search);

	const apiQuery = new URLSearchParams(currentParams); // API query parameters
	if (currentParams.get('update') === "true") {
		apiQuery.delete("update");
		apiQuery.delete("tagnumber");
		apiQuery.delete("system_serial");
	}

	if (csvDownload) {
		window.location.href = `/api/inventory?csv=true&${apiQuery.toString()}`;
		return null;
	}

	try {
		const jsonResponse: InventoryRow[] = await fetchData(`/api/inventory?${apiQuery.toString()}`, false);
		if (!jsonResponse) console.warn("No data returned from /api/inventory");
		return jsonResponse;
	} catch (error) {
		console.warn("Error fetching inventory data:", error);
		return null;
	}
}

async function fetchAllManufacturersAndModels(purgeCache: boolean = false): Promise<Array<ManufacturersAndModels> | []> {
	const cached = sessionStorage.getItem("uit_manufacturers_and_models");

  try {
		if (cached && !purgeCache) {
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

async function populateManufacturerSelect(selectEl: HTMLSelectElement, resetEl: HTMLElement, purgeCache: boolean = false) {
  if (!selectEl || !resetEl) return;

	const initialValue = selectEl.value ? selectEl.value : (new URLSearchParams(window.location.search).get('system_manufacturer') || '');
	if (initialValue !== '' && initialValue.trim().length > 0) {
		filterManufacturer.value = initialValue;
		filterModel.disabled = false;
		handleAdvSearchInputChange(selectEl, resetEl);
	} else {
		handleAdvSearchInputChange(selectEl, resetEl);
		resetSelectElement(filterModel, 'Model', true);
	}

	selectEl.disabled = true; // temporary while we update it

	try {
  	const data: ManufacturersAndModels[] = await fetchAllManufacturersAndModels(purgeCache);
		if (!data || !Array.isArray(data) || data.length === 0) throw new Error('No data returned from /api/models');

		// Sort manufacturers array - get unique key
		const manufacturerMap = new Map<string, ManufacturersAndModels>();
		for (const item of data) {
			if (!item.system_manufacturer) continue;
			if (!manufacturerMap.has(item.system_manufacturer)) {
				manufacturerMap.set(item.system_manufacturer, item);
			}
		}
		const uniqueManufacturerArr: ManufacturersAndModels[] = Array.from(manufacturerMap.values());
		uniqueManufacturerArr.sort((a, b) => {
			const manufacturerA = a.system_manufacturer;
			const manufacturerB = b.system_manufacturer;
			return manufacturerA.localeCompare(manufacturerB);
		});

		// Clear and rebuild manufacturer select options
		resetSelectElement(selectEl, 'Manufacturer');

		// Sort by formatted name
		for (const item of uniqueManufacturerArr) {
			if (!item.system_manufacturer) console.warn("Missing system_manufacturer in uniqueManufacturerArr:", item);
			const option = document.createElement('option');
			option.value = item.system_manufacturer;
			option.textContent = `${item.system_manufacturer} (${item.system_manufacturer_count || 0})`;
			selectEl.appendChild(option);
		}

		const newValue = (initialValue && uniqueManufacturerArr.some(item => item.system_manufacturer === initialValue)) ? initialValue : '';
		selectEl.value = newValue;

		if (newValue) {
			setURLParameter('system_manufacturer', selectEl.value);
		} else {
			setURLParameter('system_manufacturer', null);
		}
	} catch (error) {
		console.error('Error fetching manufacturers and models:', error);
	} finally {
		selectEl.disabled = false;
	}
}

async function populateModelSelect(selectEl: HTMLSelectElement, resetEl: HTMLElement, purgeCache: boolean = false) {
  if (!selectEl || !resetEl || !filterManufacturer || !filterManufacturerReset) {
		console.warn("Model select element or reset button not found.");
		return;
	};
	
	const initialValue = selectEl.value ? selectEl.value : (new URLSearchParams(window.location.search)).get('system_model') || '';

	if (!filterManufacturer || filterManufacturer.value === '' || filterManufacturer.value.trim().length === 0) {
		// Reset model if no manufacturer is selected
		resetSelectElement(selectEl, 'Model', true);
		handleAdvSearchInputChange(filterManufacturer, filterManufacturerReset);
		handleAdvSearchInputChange(selectEl, resetEl);
		return;
	}

	try {
		const data: ManufacturersAndModels[] = await fetchAllManufacturersAndModels(purgeCache);
		if (!data || !Array.isArray(data) || data.length === 0) return;

		data.sort((a, b) => {
			const modelA = a.system_model;
			const modelB = b.system_model;
			return modelA.localeCompare(modelB);
		});

		const filteredData = data.filter(item => item.system_manufacturer === filterManufacturer.value);

		resetSelectElement(selectEl, 'Model');
		resetEl.style.display = 'none';

		for (const item of filteredData) {
			if (!item.system_model) console.warn("Missing system_model in filteredData:", item);
			const option = document.createElement('option');
			option.value = item.system_model;
			option.textContent = item.system_model + ` (${item.system_model_count || 0})`;
			selectEl.appendChild(option);
		}

		const newValue = (initialValue && filteredData.some(item => item.system_model === initialValue)) ? initialValue : '';
		selectEl.value = newValue || '';
	} catch (error) {
		console.error('Error fetching manufacturers and models:', error);
		return;
	} finally {
		handleAdvSearchInputChange(selectEl, resetEl);
		handleAdvSearchInputChange(filterManufacturer, filterManufacturerReset);
		selectEl.disabled = false;
	}
}

async function fetchDomains(purgeCache: boolean = false): Promise<Array<Domain> | []> {
	const cached = sessionStorage.getItem("uit_domains");

	try {
		if (cached && !purgeCache) {
			const cacheEntry: DomainCache = JSON.parse(cached);
			if (Date.now() - cacheEntry.timestamp < 300000 && Array.isArray(cacheEntry.domains)) {
				console.log("Loaded domains from cache");
				return cacheEntry.domains;
			}
		}
		const data: Array<Domain> = await fetchData('/api/domains');
		if (!data || !Array.isArray(data) || data.length === 0) {
			throw new Error('No data returned from /api/domains');
		}
		const cacheEntry: DomainCache = {
			timestamp: Date.now(),
			domains: data
		};
		sessionStorage.setItem("uit_domains", JSON.stringify(cacheEntry));
		console.log("Cached domains data");
		return data;
	} catch (error) {
		console.error('Error fetching domains:', error);
		return [];
	}
}

async function populateDomainSelect(el: HTMLSelectElement, purgeCache: boolean = false) {
	if (!el) return;

	const initialValue = el.value;

	el.disabled = true;

	try {
		const domainData: Array<Domain> = await fetchDomains(purgeCache);
		if (!domainData || !Array.isArray(domainData) || domainData.length === 0) {
			throw new Error('No data returned from /api/domains');
		}

		domainData.sort((a, b) => {
			return a.domain_sort_order - b.domain_sort_order;
		});

		resetSelectElement(el, 'Domain', false, undefined);

		for (const domain of domainData) {
			const option = document.createElement('option');
			option.value = domain.ad_domain;
			option.textContent = domain.ad_domain_formatted || domain.ad_domain;
			el.appendChild(option);
		}

		el.value = (initialValue && domainData.some(item => item.ad_domain === initialValue)) ? initialValue : '';
	} catch (error) {
		console.error('Error fetching domains:', error);
	} finally {
		el.disabled = false;
	}
}

async function populateDepartmentSelect(el: HTMLSelectElement, purgeCache: boolean = false) {
	if (!el) return;

	const initialValue = el.value;

	el.disabled = true;

	try {
		const departmentsData: Array<Department> = await fetchDepartments(purgeCache);
		if (!departmentsData || !Array.isArray(departmentsData) || departmentsData.length === 0) {
			throw new Error('No data returned from /api/departments');
		}

		resetSelectElement(el, 'Department', false, undefined);

		departmentsData.sort((a, b) => {
			return a.organization_sort_order - b.organization_sort_order;
		});

		for (const department of new Set(departmentsData.map(dep => dep.organization_name_formatted || dep.organization_name))) {
			const orgEl = document.createElement('optgroup');
			orgEl.label = department ? department.trim() : 'N/A';
			el.appendChild(orgEl);
		}

		departmentsData.sort((a, b) => {
			return a.department_name_formatted.localeCompare(b.department_name_formatted);
			// return b.department_sort_order - a.department_sort_order;
		});

		for (const department of departmentsData) {
			const option = document.createElement('option');
			option.value = department.department_name;
			option.textContent = department.department_name_formatted || department.department_name;
			const parentOptGroup = Array.from(el.getElementsByTagName('optgroup')).find(group => group.label === (department.organization_name_formatted ? department.organization_name_formatted : (department.organization_name ? department.organization_name : 'N/A')));
			if (parentOptGroup) {
				parentOptGroup.appendChild(option);
			} else {
				el.appendChild(option);
			}
		}
		el.value = (initialValue && departmentsData.some(item => initialValue === item.department_name || initialValue === item.department_name_formatted)) ? initialValue : '';
	} catch (error) {
		console.error('Error fetching departments:', error);
	} finally {
		el.disabled = false;
	}
}

async function fetchStatuses(purgeCache: boolean = false): Promise<Array<Status> | []> {
	const cached = sessionStorage.getItem("uit_statuses");

	try {
		if (cached && !purgeCache) {
			const cacheEntry: StatusCache = JSON.parse(cached);
			if (Date.now() - cacheEntry.timestamp < 300000 && Array.isArray(cacheEntry.statuses)) {
				console.log("Loaded statuses from cache");
				return cacheEntry.statuses;
			}
		}
		const data: Array<Status> = await fetchData('/api/all_statuses');
		if (!data || !Array.isArray(data) || data.length === 0) {
			throw new Error('No data returned from /api/all_statuses');
		}
		const cacheEntry: StatusCache = {
			timestamp: Date.now(),
			statuses: data
		};
		sessionStorage.setItem("uit_statuses", JSON.stringify(cacheEntry));
		console.log("Cached statuses data");
		return data;
	} catch (error) {
		console.error('Error fetching statuses:', error);
		return [];
	}
}

async function populateStatusSelect(el: HTMLSelectElement, purgeCache: boolean = false) {
	if (!el) return;

	const initialValue = el.value;

	el.disabled = true;

	try {
		const statusData: Array<Status> = await fetchStatuses(purgeCache);
		if (!statusData || !Array.isArray(statusData) || statusData.length === 0) {
			throw new Error('No data returned from /api/statuses');
		}

		statusData.sort((a, b) => {
			return a.sort_order - b.sort_order;
		});

		resetSelectElement(el, 'Status', false, undefined);


		for (const status of statusData) {
			const option = document.createElement('option');
			option.value = status.status;
			option.textContent = status.status_formatted || status.status;
			el.appendChild(option);
		}
		el.value = (initialValue && statusData.some(item => initialValue === item.status || initialValue === item.status_formatted)) ? initialValue : '';
	} catch (error) {
		console.error('Error fetching statuses:', error);
	} finally {
		el.disabled = false;
	}
}

async function populateLocationSelect(el: HTMLSelectElement, purgeCache: boolean = false) {
	if (!el) return;
	const initialValue = el.value;

	el.disabled = true;
	try {
		const locationData: Array<AllLocations> = await fetchAllLocations(purgeCache);
		if (!locationData || !Array.isArray(locationData) || locationData.length === 0) {
			throw new Error('No data returned from /api/locations (populateLocationSelect)');
		}
		locationData.sort((a, b) => { // alpahbetical here, not by timestamp
			const locA = a.location_formatted || a.location || '';
			const locB = b.location_formatted || b.location || '';
			return locA.localeCompare(locB);
		});

		resetSelectElement(el, 'Location', false, undefined);

		for (const location of locationData) {
			const option = document.createElement('option');
			option.value = location.location || '';
			option.textContent = location.location_formatted || location.location || 'N/A' + (location.location_count !== null ? ` (${location.location_count})` : '');
			el.appendChild(option);
		}

		el.value = (initialValue && locationData.some(item => initialValue === item.location || initialValue === item.location_formatted)) ? initialValue : '';

		el.disabled = false;
	}	catch (error) {
		console.error('Error fetching locations:', error);
	}
}
		

inventoryFilterForm.addEventListener("submit", (event) => {
  event.preventDefault();
  renderInventoryTable();
});

advSearchFormReset.addEventListener("click", async (event) => {
  event.preventDefault();

	for (const param of advSearchParams) {
		if (!param.inputElement || !param.paramString) continue;
		param.inputElement.value = '';
		handleAdvSearchInputChange(param.inputElement, param.resetElement);
	}

	try{
		await Promise.all([
			populateDepartmentSelect(filterDepartment),
			populateLocationSelect(filterStatus),
			populateManufacturerSelect(filterManufacturer, filterManufacturerReset).then(() => populateModelSelect(filterModel, filterModelReset)),
			populateDomainSelect(filterDomain),
			renderInventoryTable(),
		]);
	} catch (error) {
		console.error("Error resetting filters and fetching data:", error);
	}
});