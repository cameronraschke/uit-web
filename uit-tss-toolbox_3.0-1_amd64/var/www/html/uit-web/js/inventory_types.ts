type AllDomainsRow = {
	ad_domain: string;
	ad_domain_formatted: string;
	domain_sort_order: number;
	client_count: number;
};

type DomainCache = {
	timestamp: number;
	domains: AllDomainsRow[];
};

type AllManufacturersAndModelsRow = {
	system_manufacturer: string;
	system_model: string;
	system_model_count: number;
	system_manufacturer_count?: number;
};

type ManufacturerAndModelsCache = {
	timestamp: number;
	manufacturers_and_models: AllManufacturersAndModelsRow[];
};

type Statuses = {
	status: string,
	status_formatted: string,
	status_sort_order: number,
	status_type: string;
	client_count: number;
};

type StatusCache = {
	timestamp: number;
	statuses: Record<string, Statuses[]>;
};

type AdvSearchFilterElement = {
	inputElement: HTMLSelectElement;
	negationElement: HTMLInputElement;
	resetElement: HTMLElement;
};

type AdvSearchOptionString = {
	param_value: string | null;
	not: boolean | null;
};

type BulkUpdateRequest = {
	bulk_location: string | null;
	bulk_tagnumbers: number[];
};

type InventoryForm = {
	time: string | null;
	tagnumber: number | null;
	system_serial: string | null;
	location: string | null;
	building: string | null;
	room: string | null;
	system_manufacturer: string | null;
	system_model: string | null;
	device_type: string | null;
	department_name: string | null;
	ad_domain: string | null;
	property_custodian: string | null;
	acquired_date: Date | null;
	retired_date: Date | null;
	is_broken: boolean | null;
	disk_removed: boolean | null;
	last_hardware_check: Date | null;
	status: string | null;
	checkout_bool: boolean | null;
	checkout_date: Date | null;
	return_date: Date | null;
	note: string | null;
	file_count: number | null;
};

type Department = {
	department_name: string;
	department_name_formatted: string;
	department_sort_order: number;
	organization_name: string;
	organization_name_formatted: string;
	organization_sort_order: number;
	client_count: number;
};

type DepartmentsCache = {
	timestamp: number;
	departments: Department[];
};

type ClientLookupResult = {
	tagnumber: number | null;
	system_serial: string | null;
};

type AllLocations = {
	timestamp: Date | null;
	location: string | null;
	location_formatted: string | null;
	location_count: number | null;
};

type AllLocationsCache = {
	timestamp: number;
	locations: AllLocations[];
};

type InventoryTableRow = {
	tagnumber: number | 0;
	system_serial: string | "";
	location_formatted: string | "";
	building: string | "";
	room: string | "";
	system_manufacturer: string | "";
	system_model: string | "";
	device_type: string | "";
	device_type_formatted: string | "";
	department_formatted: string | "";
	ad_domain_formatted: string | "";
	status_formatted: string | "";
	is_broken: boolean | null;
	note: string | "";
	last_updated: string | "";
	file_count: number | null;
	client_configuration_errors: string[] | null;
};

const inventoryFilterForm = document.getElementById('adv-search-form') as HTMLFormElement;
const advSearchFormReset = document.getElementById('adv-search-form-reset-button') as HTMLElement;
const advSearchLocation = document.getElementById('adv-search-location') as HTMLSelectElement;
const advSearchLocationNegation = document.getElementById('adv-search-location-negation') as HTMLInputElement;
const advSearchLocationReset = document.getElementById('adv-search-location-reset') as HTMLElement;
const filterDepartment = document.getElementById('adv-search-department') as HTMLSelectElement;
const filterDepartmentNegation = document.getElementById('adv-search-department-negation') as HTMLInputElement;
const filterDepartmentReset = document.getElementById('adv-search-department-reset') as HTMLElement;
const filterManufacturer = document.getElementById('adv-search-manufacturer') as HTMLSelectElement;
const filterManufacturerNegation = document.getElementById('adv-search-manufacturer-negation') as HTMLInputElement;
const filterManufacturerReset = document.getElementById('adv-search-manufacturer-reset') as HTMLElement;
const filterModel = document.getElementById('adv-search-model') as HTMLSelectElement;
const filterModelNegation = document.getElementById('adv-search-model-negation') as HTMLInputElement;
const filterModelReset = document.getElementById('adv-search-model-reset') as HTMLElement;
const filterDomain = document.getElementById('adv-search-ad-domain') as HTMLSelectElement;
const filterDomainNegation = document.getElementById('adv-search-ad-domain-negation') as HTMLInputElement;
const filterDomainReset = document.getElementById('adv-search-ad-domain-reset') as HTMLElement;
const filterStatus = document.getElementById('adv-search-status') as HTMLSelectElement;
const filterStatusNegation = document.getElementById('adv-search-status-negation') as HTMLInputElement;
const filterStatusReset = document.getElementById('adv-search-status-reset') as HTMLElement;
const filterBroken = document.getElementById('adv-search-is-broken') as HTMLSelectElement;
const filterBrokenNegation = document.getElementById('adv-search-is-broken-negation') as HTMLInputElement;
const filterBrokenReset = document.getElementById('adv-search-is-broken-reset') as HTMLElement;
const filterHasImages = document.getElementById('adv-search-has-images') as HTMLSelectElement;
const filterHasImagesNegation = document.getElementById('adv-search-has-images-negation') as HTMLInputElement;
const filterHasImagesReset = document.getElementById('adv-search-has-images-reset') as HTMLElement;
const filterDeviceType = document.getElementById('adv-search-device-type') as HTMLSelectElement;
const filterDeviceTypeNegation = document.getElementById('adv-search-device-type-negation') as HTMLInputElement;
const filterDeviceTypeReset = document.getElementById('adv-search-device-type-reset') as HTMLElement;

const advSearchParams: Record<string, AdvSearchFilterElement> = {
	'filter_location': { inputElement: advSearchLocation, negationElement: advSearchLocationNegation, resetElement: advSearchLocationReset },
	'filter_system_manufacturer': { inputElement: filterManufacturer, negationElement: filterManufacturerNegation, resetElement: filterManufacturerReset },
	'filter_system_model': { inputElement: filterModel, negationElement: filterModelNegation, resetElement: filterModelReset },
	'filter_device_type': { inputElement: filterDeviceType, negationElement: filterDeviceTypeNegation, resetElement: filterDeviceTypeReset },
	'filter_department_name': { inputElement: filterDepartment, negationElement: filterDepartmentNegation, resetElement: filterDepartmentReset },
	'filter_ad_domain': { inputElement: filterDomain, negationElement: filterDomainNegation, resetElement: filterDomainReset },
	'filter_status': { inputElement: filterStatus, negationElement: filterStatusNegation, resetElement: filterStatusReset },
	'filter_is_broken': { inputElement: filterBroken, negationElement: filterBrokenNegation, resetElement: filterBrokenReset },
	'filter_has_images': { inputElement: filterHasImages, negationElement: filterHasImagesNegation, resetElement: filterHasImagesReset },
};



// Table elements
const inventoryTableBody = document.getElementById('inventory-table-body') as HTMLTableSectionElement;
const inventoryTableRowCountEl = document.getElementById('inventory-table-rowcount') as HTMLElement;
const inventoryTableSearch = document.getElementById('inventory-table-search') as HTMLInputElement;
const inventoryTableSortBy = document.getElementById('inventory-table-sort-by') as HTMLSelectElement;
let inventoryTableSearchDebounce: ReturnType<typeof setTimeout> | null = null;
let activePortalTooltip: HTMLDivElement | null = null;
let hasPortalTooltipGlobalListeners = false;

let updatingInventory = false;



// Bulk update form 
const toggleBulkUpdate = document.querySelector('#inventory-toggle-bulk') as HTMLButtonElement;
const bulkUpdateForm = document.querySelector('#inventory-bulk-update') as HTMLFormElement;
const bulkUpdateLocationInput = document.querySelector('#bulk_location') as HTMLInputElement;
const bulkUpdateTagInput = document.querySelector('#bulk_tagnumbers') as HTMLInputElement;
const bulkUpdateSubmitButton = document.querySelector('#inventory-bulk-update-submit') as HTMLButtonElement;
const bulkUpdateCancelButton = document.querySelector('#inventory-bulk-update-cancel') as HTMLButtonElement;

// Inventory lookup form elements
const clientLookupForm = document.querySelector('#inventory-lookup-form') as HTMLFormElement;
const clientLookupWarningMessage = document.getElementById('existing-inventory-message') as HTMLElement;
const clientLookupTagInput = document.getElementById('inventory-tag-lookup') as HTMLInputElement;
const clientLookupSerial = document.getElementById('inventory-serial-lookup') as HTMLInputElement;
const clientLookupSubmitButton = document.getElementById('inventory-lookup-submit-button') as HTMLButtonElement;
const clientMoreDetails = document.getElementById('inventory-lookup-more-details') as HTMLButtonElement;
const clientViewPhotos = document.getElementById('inventory-lookup-photo-album') as HTMLButtonElement;
const clientAddPhotos = document.getElementById('inventory-lookup-add-photos') as HTMLButtonElement;
const allTagsDatalist = document.getElementById('inventory-tag-suggestions') as HTMLDataListElement;
const csvDownloadButton = document.getElementById('adv-search-download-csv') as HTMLButtonElement;
const printCheckoutLink = document.getElementById('print-checkout-link') as HTMLElement;
const printCheckoutContainer = document.getElementById('print-checkout-container') as HTMLElement;

// Inventory update form elements
const formAnchor = document.querySelector('#update-and-search-container') as HTMLElement;
const updateForm = document.getElementById('inventory-update-form') as HTMLFormElement;
const lastUpdateTime = document.getElementById('last-update-time-message') as HTMLElement;
const locationEl = document.getElementById('location') as HTMLInputElement;
const buildingUpdate = document.querySelector("#building") as HTMLInputElement;
const roomUpdate = document.querySelector("#room") as HTMLInputElement;
const manufacturerUpdate = document.querySelector("#system_manufacturer") as HTMLInputElement;
const modelUpdate = document.querySelector("#system_model") as HTMLInputElement;
const deviceTypeUpdate = document.querySelector("#device_type") as HTMLSelectElement;
const departmentEl = document.getElementById('department_name') as HTMLSelectElement;
const adDomainUpdate = document.querySelector("#ad_domain") as HTMLSelectElement;
const propertyCustodianUpdate = document.querySelector("#property_custodian") as HTMLInputElement;
const acquiredDateUpdate = document.querySelector("#acquired_date") as HTMLInputElement;
const retiredDateUpdate = document.querySelector("#retired_date") as HTMLInputElement;
const isBrokenUpdate = document.querySelector("#is_broken") as HTMLSelectElement;
const diskRemovedUpdate = document.querySelector("#disk_removed") as HTMLSelectElement;
const lastHardwareCheckUpdate = document.querySelector("#last_hardware_check") as HTMLInputElement;
const clientStatusUpdate = document.querySelector("#status") as HTMLSelectElement;
const checkoutBoolUpdate = document.querySelector("#checkout_bool") as HTMLSelectElement;
const checkoutDateUpdate = document.querySelector("#checkout_date") as HTMLInputElement;
const returnDateUpdate = document.querySelector("#return_date") as HTMLInputElement;
const noteUpdate = document.querySelector("#note") as HTMLInputElement;
const fileInputUpdate = document.querySelector("#inventory-file-input") as HTMLInputElement;
const submitUpdate = document.getElementById('inventory-update-submit-button') as HTMLButtonElement;
const cancelUpdate = document.getElementById('inventory-update-cancel-button') as HTMLButtonElement;

// Show/hide parts of form
const showLocationPart = document.querySelector("#show-location-data") as HTMLButtonElement;
const showHardwarePart = document.querySelector("#show-hardware-data") as HTMLButtonElement;
const showSoftwarePart = document.querySelector("#show-software-data") as HTMLButtonElement;
const showPropertyPart = document.querySelector("#show-property-data") as HTMLButtonElement;
const showNotesFilesPart = document.querySelector("#show-notes-files-data") as HTMLButtonElement;

const locationPart = document.querySelectorAll("[data-location-part]") as NodeListOf<HTMLDivElement>;
const hardwarePart = document.querySelectorAll("[data-hardware-part]") as NodeListOf<HTMLDivElement>;
const softwarePart = document.querySelectorAll("[data-software-part]") as NodeListOf<HTMLDivElement>;
const propertyPart = document.querySelectorAll("[data-property-part]") as NodeListOf<HTMLDivElement>;
const notesFilesPart = document.querySelectorAll("[data-note-files-part]") as NodeListOf<HTMLDivElement>;

const locationFormShowSections = [showLocationPart, showHardwarePart, showSoftwarePart, showPropertyPart, showNotesFilesPart];

const allowedFileNameRegex = /^[a-zA-Z0-9.\-_ ()]+\.(jpg|jpeg|jfif|png|mp4)$/i; // file name + extension
const allowedFileExtensions = [".jpg", ".jpeg", ".jfif", ".png", ".mp4"];

const statusesThatIndicateBroken = ["needs-repair"];
const statusesThatIndicateCheckout = ["checked-out", "reserved-for-checkout"];

const allInventoryUpdateFields = [
	clientLookupTagInput,
	clientLookupSerial,
	locationEl,
	buildingUpdate,
	roomUpdate,
	manufacturerUpdate,
	modelUpdate,
	deviceTypeUpdate,
	departmentEl,
	adDomainUpdate,
	propertyCustodianUpdate,
	acquiredDateUpdate,
	retiredDateUpdate,
	isBrokenUpdate,
	diskRemovedUpdate,
	lastHardwareCheckUpdate,
	clientStatusUpdate,
	checkoutBoolUpdate,
	checkoutDateUpdate,
	returnDateUpdate,
	noteUpdate,
	fileInputUpdate,
];

const requiredInventoryUpdateFields = [
	clientLookupTagInput,
	clientLookupSerial,
	locationEl,
	departmentEl,
	adDomainUpdate,
	clientStatusUpdate
];

const buttonsVisibleWhenUpdating = [
	clientMoreDetails,
	clientViewPhotos,
	clientAddPhotos,
];



function updateURLFromAdvFilters(): void {
	for (const paramName in advSearchParams) {
		const param = advSearchParams[paramName];
		if (!param.inputElement) continue;
		if (param.inputElement.value && param.inputElement.value.trim().length > 0) {
			const urlValue = {
				param_value: param.inputElement.value,
				not: (param.negationElement && param.negationElement.checked === true) ? true : null,
			};
			setURLParameter(paramName, JSON.stringify(urlValue), true);
		} else {
			setURLParameter(paramName, null);
		}
	}
}