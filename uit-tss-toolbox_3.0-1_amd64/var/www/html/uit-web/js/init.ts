interface Window {
  allTags: number[];
}

type TagCache = {
	tags: number[];
	timestamp: number;
};

type AuthStatusResponse = {
	status: string | null;
	time: string | null;
};

type DeviceType = {
	device_type: string | null;
	device_type_formatted: string | null;
	device_meta_category: string | null;
	sort_order: string | null;
};

type DeviceTypeCache = {
	deviceTypes: DeviceType[] | [];
	timestamp: number;
};

const inputCSSClasses = ["empty-input", "empty-required-input", "changed-input", "readonly-input"];

function truncateString(inputStr: string, maxLength: number): { truncatedString: string, isTruncated: boolean } {
	const str = inputStr ? inputStr.trim() : "";
	if (str.length <= maxLength) {
		return { truncatedString: inputStr, isTruncated: false };
	}
	const newStr = str.substring(0, maxLength - 3) + "...";
	return { truncatedString: newStr, isTruncated: true };
}

function createTextCell(elID: string | undefined, datasetKey: string | undefined, inputStr: string | null, truncateLen: number | undefined, customError: string | undefined) : HTMLTableCellElement {
	const cell = document.createElement('td');

	if (elID) cell.id = elID;	

	if (inputStr) {
		if (datasetKey) cell.dataset[`${datasetKey}`] = inputStr || '';
		if (truncateLen && inputStr.length > truncateLen) {
			const truncated = truncateString(inputStr, truncateLen);
			cell.textContent = truncated.truncatedString;
			cell.title = inputStr + " (click to expand)";
			cell.style.cursor = 'pointer';
			cell.addEventListener('click', () => {
				cell.textContent = inputStr;
				if (cell.title) cell.removeAttribute('title');
				cell.style.cursor = 'auto';
			}, { once: true });
		} else {
			cell.textContent = inputStr;
		}
	} else {
		cell.style.fontStyle = 'italic';
		cell.textContent = customError !== undefined ? customError : 'N/A';
	}
	return cell;
}

// Boolean broken status
function createBooleanCell(elID: string | undefined, datasetKey: string | undefined, inputBool: boolean | null, trueText: string | undefined, falseText: string | undefined, customError: string | undefined) : HTMLTableCellElement {
  const cell = document.createElement('td');

	if (elID) cell.id = elID;
  
  if (typeof inputBool === 'boolean') {
		if (datasetKey) cell.dataset[`${datasetKey}`] = inputBool !== undefined ? String(inputBool) : '';
    cell.textContent = inputBool ? trueText || 'true' : falseText || 'false';
	} else {
		cell.style.fontStyle = 'italic';
    cell.textContent = customError !== undefined ? customError : 'N/A';
  }
  return cell;
}

// Date formatting
function createTimestampCell(elID: string | undefined, datasetKey: string | undefined, inputStr: string | null, customError: string | undefined) : HTMLTableCellElement {
  const cell = document.createElement('td');

	if (elID) cell.id = elID;
  
  if (!inputStr || inputStr.trim().length === 0) {
		cell.style.fontStyle = 'italic';
    cell.textContent = customError !== undefined ? customError : 'N/A';
    return cell;
  }
  
  const date = new Date(inputStr);
  
  if (!isNaN(date.getTime())) {
    const formatted = `${date.toLocaleDateString()} ${date.toLocaleTimeString()}`;
    if (datasetKey) cell.dataset[`${datasetKey}`] = formatted;
    cell.textContent = formatted;
  } else {
		cell.style.fontStyle = 'italic';
		cell.textContent = customError !== undefined ? customError : 'N/A';
	}
  return cell;
}

function getInputStringValue(inputEl: HTMLInputElement | HTMLSelectElement): string | null {
	if (!inputEl) {
		throw new Error("Input element not found in DOM");
	}
	const value = inputEl.value ? inputEl.value.trim() : null;
	if (inputEl.required && (!value || value.length === 0)) {
		throw new Error(`${inputEl.id} field cannot be empty`);
	}
	return value;
}

function getInputNumberValue(inputEl: HTMLInputElement | HTMLSelectElement): number | null {
	if (!inputEl) {
		throw new Error("Input element not found in DOM");
	}
	const value = inputEl.value ? inputEl.value.trim() : null;
	if (inputEl.required && (!value || value.length === 0)) {
		throw new Error(`${inputEl.id} field cannot be empty`);
	}
	const numValue = Number(value);
	if (isNaN(numValue)) {
		throw new Error(`${inputEl.id} field must be a valid number`);
	}
	return numValue;
}

function getInputBooleanValue(inputEl: HTMLInputElement | HTMLSelectElement): boolean | null {
	if (!inputEl) {
		throw new Error("Input element not found in DOM");
	}
	const value = inputEl.value ? inputEl.value.trim() : null;
	if (value === "true") return true;
	else if (value === "false") return false;
	else if (!inputEl.required) return null;
	else throw new Error(`${inputEl.id} field must be true or false`);
}

function getInputDateValue(inputEl: HTMLInputElement | HTMLSelectElement, isNull: boolean = false): Date | null {
	if (!inputEl) {
		throw new Error("Input element not found in DOM");
	}
	const value = inputEl.value ? inputEl.value.trim() : null;
	if (inputEl.required && (!value || value.length === 0)) {
		throw new Error(`${inputEl.id} field cannot be empty`);
	}

	const dateObj = new Date(value + "T00:00:00");
	if (isNaN(dateObj.getTime()) && !(isNull && (!value || value.length === 0))) {
		throw new Error(`${inputEl.id} field must be a valid date`);
	}

	if (isNull && (!value || value.length === 0)) {
		return null;
	}
	return dateObj;
}
function getInputTimeValue(inputEl: HTMLInputElement | HTMLSelectElement): Date | null {
	if (!inputEl) throw new Error("Input element not found in DOM");

	const value = inputEl.value ? inputEl.value.trim() : null;
	if (inputEl.required && (!value || value.length === 0)) throw new Error(`${inputEl.id} field cannot be empty`);
	if (!value || value.length === 0) return null;

	const timeObj = new Date(value);
	if (isNaN(timeObj.getTime())) throw new Error(`${inputEl.id} field must be a valid datetime`);
	return timeObj;
}

async function checkAuthStatus(): Promise<boolean> {
	// server should auto redirect, but only on page refresh
	// this function redirects w/o needing a refresh
	if (window.location.pathname === '/login' || window.location.pathname === '/logout') {
		return true;
	}
	if (!navigator.onLine) {
		console.warn("Offline, skipping auth check");
		return true;
	}
	try {
		const url = "/api/check_auth";
		const response: AuthStatusResponse = await fetchData(url, false);
		if (response && response.status === "authenticated") {
			return true;
		} else {
			return false;
		}
	} catch (error) {
		console.error("Error while checking authentication status, redirecting:", error);
		window.location.href = "/logout";
		return false;
	}
}

// document.body.addEventListener("click", async (event) => {
// 	if (window.location.pathname === '/login' || window.location.pathname === '/logout') {
// 		return;
// 	}
// 	const target = event.target as HTMLElement;
// 	// if (target && target.matches(".requires-auth, .requires-auth *")) {
// 	if (target) {
// 		const isAuthenticated = await checkAuthStatus();
// 		if (!isAuthenticated) {
// 			event.preventDefault();
// 			window.location.href = "/logout";
// 		}
// 	}
// });

const checkAuthTimeout = 5000; // 5 seconds
let authCheckTimeout: number;
authCheckTimeout = setInterval(async () => {
	if (document.visibilityState !== "visible") return;

	const isAuthenticated = await checkAuthStatus();
	if (!isAuthenticated) {
		window.location.href = "/logout";
	}
}, checkAuthTimeout);

document.addEventListener("visibilitychange", async () => {
	if (window.location.pathname === '/login' || window.location.pathname === '/logout') {
		return;
	}
	// Clear any existing timeouts
	clearInterval(authCheckTimeout);

	if (document.visibilityState === "visible") {
		
		// Immediately check auth status
		const isAuthenticated = await checkAuthStatus();
		if (!isAuthenticated) {
			window.location.href = "/logout";
			return;
		}

		// Don't proceed to authCheckTimeout if tab is no longer visible, otherwise the timeout will still run
		// Auth status will be checked again on next visibilitychange
		if (document.visibilityState !== "visible") {
			return;
		}

		// Delayed check
		authCheckTimeout = setInterval(async () => {
			if (document.visibilityState !== "visible") return;

			const isStillAuthenticated = await checkAuthStatus();
			if (!isStillAuthenticated) {
				window.location.href = "/logout";
			}
		}, checkAuthTimeout);
	}
});

function removeCSSClasses(el: HTMLElement, ...classesToRemove: string[]) {
	const currentClasses = Array.from(el.classList);
	if (classesToRemove && classesToRemove.length > 0) {
		for (const className of classesToRemove) {
			el.classList.remove(className);
		}
		return;
	}
	for (const className of currentClasses) {
		el.classList.remove(className);
	}
}

function resetSelectElement(selectElement: HTMLSelectElement, defaultText: string, isDisabled: boolean = false, newCSSClass = "") {
	selectElement.disabled = true;
	selectElement.innerHTML = "";
	removeCSSClasses(selectElement, ...inputCSSClasses);
	const defaultOption = document.createElement('option');
	defaultOption.value = "";
	defaultOption.textContent = defaultText;
	selectElement.required = false;
	selectElement.disabled = isDisabled;
	defaultOption.selected = true;
	// defaultOption.hidden = true;
	selectElement.appendChild(defaultOption);
	if (newCSSClass && newCSSClass.trim().length > 0) {
		selectElement.classList.add(newCSSClass);
	}
	selectElement.addEventListener('click', () => {
		defaultOption.disabled = true;
	});
	selectElement.addEventListener('focus', () => {
		selectElement.style.backgroundColor = "var(--fg-color)";
	});
	selectElement.addEventListener('blur', () => {
		selectElement.style.removeProperty("background-color");
	});
}

function resetInputElement(inputElement: HTMLInputElement, placeholderText: string, isReadOnly: boolean = false, newCSSClass: string | undefined) {
	inputElement.disabled = true;
	removeCSSClasses(inputElement, ...inputCSSClasses);
	inputElement.value = "";
	inputElement.placeholder = placeholderText;
	inputElement.readOnly = isReadOnly;
	inputElement.required = false;
	inputElement.disabled = false;
	if (newCSSClass && newCSSClass.trim().length > 0) {
		inputElement.classList.add(newCSSClass);
	}
}

function validateTagInput(tagInput: number): boolean {
	let validRange = false;
	let validRegex = false;
	const regexPattern = /^[0-9]{6}$/;

	if (tagInput > 1 && tagInput < 999999) validRange = true;
	if (regexPattern.test(tagInput.toString())) validRegex = true;
	return validRange && validRegex;
}

function jsonToBase64(jsonString: string) {
	try {
		const jsonParsed: any = JSON.parse(jsonString);
		if (!jsonParsed) {
			throw new TypeError("Input is not a valid JSON string");
		}
		if (jsonParsed && typeof jsonParsed === 'object' && Object.prototype.hasOwnProperty.call(jsonParsed, '__proto__')) {
			throw new Error(`Prototype pollution detected`);
		}

		const utf8Bytes = new TextEncoder().encode(jsonString);
		const binaryStr = Array.from(utf8Bytes, (byte: number) => String.fromCharCode(byte)).join("");
		const base64JsonData: string = btoa(binaryStr).replace(/\+/g, "-").replace(/\//g, "_").replace(/=+$/, "");
		// Decode json with base64ToJson and double-check that it's correct.
		const decodedJson = base64ToJson(base64JsonData);
		if (!base64JsonData || JSON.stringify(jsonParsed) !== decodedJson) {
			throw new Error(`Encoded json does not match decoded json. \n${base64JsonData}\n${decodedJson}`);
		}
		return base64JsonData;
	} catch (error) {
		console.error("Invalid JSON string:", error);
		return null;
	}
}

function base64ToJson(inputStr: string) {
	try {
		if (typeof inputStr !== 'string') {
			throw new TypeError("Input is not a valid base64 string");
		}
		if (inputStr.trim() === "") {
			throw new Error("Base64 string is empty");
		}

		const standardBase64 = inputStr.replace(/-/g, '+').replace(/_/g, '/');
		const pad = standardBase64.length % 4;
		const paddedBase64 = pad ? standardBase64 + "====".slice(0, 4 - pad) : standardBase64;

		const base64Bytes: string = atob(paddedBase64);
		const byteArray = Uint8Array.from(base64Bytes, c => c.charCodeAt(0));
		const jsonString = new TextDecoder().decode(byteArray);
		const jsonParsed = JSON.parse(jsonString);
		if (!jsonParsed) {
			throw new TypeError("Input is not a valid JSON string");
		}
		if (jsonParsed && typeof jsonParsed === 'object' && Object.prototype.hasOwnProperty.call(jsonParsed, '__proto__')) {
			throw new Error(`Prototype pollution detected`);
		}
		return JSON.stringify(jsonParsed);
	} catch (error) {
		console.log("Error decoding base64: " + error);
		return null;
	}
}

function getURLParamName(filterElement: HTMLSelectElement): string {
	for (const param of advSearchParams) {
		if (param.inputElement === filterElement) {
			return param.paramString;
		}
	}
	return '';
}

function updateURLFromFilters(): void {
	for (const param of advSearchParams) {
		if (!param.inputElement || !param.paramString) continue;
		setURLParameter(param.paramString, param.inputElement.value ? param.inputElement.value : null);
	}
}

async function fetchData(url: string, returnText = false, fetchOptions: RequestInit = {}): Promise<any> {
  try {
    if (!url || url.trim().length === 0) {
      throw new Error("No URL specified for fetchData");
    }

		const headers = new Headers();
		headers.append('Content-Type', 'application/x-www-form-urlencoded');

		// Try to add bearer token, but do not fail the request if not found.
		try {
			const bearerToken = await getKeyFromIndexDB("bearerToken");
			if (bearerToken) {
				headers.append('Authorization', 'Bearer ' + bearerToken);
			}
		} catch (tokenErr) {
			console.warn("Bearer token not available; proceeding with cookies only", tokenErr);
		}
    

    const response = await fetch(url, {
      method: 'GET',
      headers: headers,
      credentials: 'same-origin',
			...fetchOptions
    });

    // No content (OPTIONS request)
    if (response.status === 204) {
      return null;
    }
    if (!response.ok) {
			if (response.status === 401 || response.status === 403) {
				console.warn("Unauthorized response from server, redirecting to logout");
				window.location.href = "/logout";
				return;
			}
      throw new Error(`Error fetching data: ${url} ${response.status}`);
    }
    // if (!response.headers || !response.headers.get('Content-Type') || !response.headers.get('Content-Type').includes('application/json')) {
    //   throw new Error('Response is undefined or not JSON');
    // }

    const textData = await response.text();
    
    if (returnText) return textData;
    if (!returnText) {
      const jsonData = await JSON.parse(textData);
      if (!jsonData || Object.keys(jsonData).length === 0 || (jsonData && typeof jsonData === 'object' && Object.prototype.hasOwnProperty.call(jsonData, '__proto__'))) {
        console.warn("Response JSON is empty: " + url);
      }
      return jsonData;
    }
  } catch (error) {
    throw error;
  }
}

function openTokenDB(): Promise<IDBDatabase> {
  return new Promise((resolve, reject) => {
    const request = indexedDB.open("uitTokens", 1);
    request.onupgradeneeded = (event: IDBVersionChangeEvent) => {
      const db = (event.target as IDBOpenDBRequest).result as IDBDatabase;
      const objectStore = db.createObjectStore("uitTokens", { keyPath: "tokenType" });
      objectStore.createIndex("authStr", "authStr", { unique: true });
      objectStore.createIndex("basicToken", "basicToken", { unique: true });
      objectStore.createIndex("bearerToken", "bearerToken", { unique: true });
    };
    request.onsuccess = event => resolve((event.target as IDBOpenDBRequest).result as IDBDatabase);
    request.onerror = event => reject("Cannot open token DB: " + (event.target as IDBOpenDBRequest).error);
  });
}

async function getKeyFromIndexDB(key: string) {
    if (!key || key.length === 0 || typeof key !== "string" || key.trim() === "") {
      throw new Error("Key is invalid: " + key);
    }

    try {
        const dbConn: IDBDatabase = await new Promise((resolve, reject) => {
            const tokenDBConnection = indexedDB.open("uitTokens", 1);
            tokenDBConnection.onsuccess = (event) => resolve((event.target as IDBOpenDBRequest).result as IDBDatabase);
            tokenDBConnection.onerror = (event) => reject("Error opening IndexedDB: " + (event.target as IDBOpenDBRequest).error);
        });

        const tokenTransaction = dbConn.transaction(["uitTokens"], "readonly");
        const tokenObjectStore = tokenTransaction.objectStore("uitTokens");

        const tokenObj: any = await new Promise((resolve, reject) => {
            const tokenRequest = tokenObjectStore.get(key);
            tokenRequest.onsuccess = event => resolve((event.target as IDBRequest).result);
            tokenRequest.onerror = event => reject("Error querying token from IndexedDB: " + (event.target as IDBRequest).error as string);
        });

        if (!tokenObj || !tokenObj.value || typeof tokenObj.value !== "string" || tokenObj.value.trim() === "") {
            throw new Error("No token found for key: " + key);
        }
        return tokenObj.value;
    } catch (error) {
        throw new Error("Error accessing IndexedDB: " + error);
    }
}

async function generateSHA256Hash(input: string) {
    if (!input || input.length === 0 || input.trim() === "") {
      throw new Error("Hash input is invalid: " + input);
    }

    const encoder = new TextEncoder();
    const encodedInput = encoder.encode(input);
    const hashBuffer = await crypto.subtle.digest("SHA-256", encodedInput);
    const hashArray = Array.from(new Uint8Array(hashBuffer));
    const hashStr = hashArray.map(b => b.toString(16).padStart(2, "0")).join("");
    if (!hashStr || hashStr.length === 0 || hashStr.trim() === "") {
      throw new Error("Hash generation failed: " + input);
    }
    return hashStr;
}

async function getTagsFromServer(): Promise<TagCache | null> {
	let tagObj: TagCache = { tags: [], timestamp: Date.now() };

	const url = "/api/all_tags";
	try {
		const data = await fetchData(url, false); // expect JSON array
		if (!data) {
			console.warn("No data returned from /api/all_tags");
			return null;
		}

		const tagArr: any = data;
		if (!Array.isArray(tagArr) || tagArr.length === 0) {
			console.warn("/api/all_tags response is not an array or is empty");
			return null;
		}
		for (const tag of tagArr) {
			const tagNum = typeof tag === "number" ? tag : Number(tag);
			if (!Number.isFinite(tagNum) || !validateTagInput(tagNum)) {
				console.warn("Invalid tag in /api/all_tags response: " + tag);
				return null;
			}
			tagObj.tags.push(tagNum);
		}
		tagObj.timestamp = Date.now();
		return tagObj;
	} catch (error) {
		console.error("Error fetching tags from server:", error);
		return null;
	}
}

function setURLParameter(urlParameter: string | null, value: string | null) {
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

function getCachedTags(): TagCache | null {
	const cached = sessionStorage.getItem("uit_all_tags");
	if (!cached) return null;

	try {
		const cacheEntry: TagCache = JSON.parse(cached);
		// 5 min cache expiry
		if (Date.now() - cacheEntry.timestamp < 300000 && Array.isArray(cacheEntry.tags)) {
			console.log("Loaded tags from cache");
			return cacheEntry;
		}
	} catch (e) {
		sessionStorage.removeItem("uit_all_tags");
	}
	return null;
}

async function getAllTags():  Promise<TagCache | null> {
	if (window.location.pathname === '/login' || window.location.pathname === '/logout') {
		return null;
	}
	const cachedTags = getCachedTags();
	if (cachedTags) return cachedTags;

	try {
		const refreshedTags = await getTagsFromServer();
		if (refreshedTags) {
			sessionStorage.setItem("uit_all_tags", JSON.stringify({
				tags: refreshedTags.tags,
				timestamp: refreshedTags.timestamp
			}));
			return refreshedTags;
		}
	} catch (e) {
		console.warn("Error parsing /api/all_tags response:", e);
		return null;
	}
	return null;
}

async function waitForNextPaint(frames = 1) {
  while (frames-- > 0) {
    await new Promise(requestAnimationFrame);
  }
}



document.addEventListener("DOMContentLoaded", async () => {
	if (!navigator.onLine) {
		console.warn("Offline, redirecting to logout");
		window.location.href = "/logout";
		return;
	}

	// Check auth status
	try {
		const firstCheck = await checkAuthStatus();
		if (firstCheck === false) {
			window.location.href = "/logout";
		}
	} catch (error) {
		console.error("Error during initial auth check:", error);
		window.location.href = "/logout";
	}

	// Load all tags
	try {
		const allTags = await getAllTags();
		if (allTags && Array.isArray(allTags.tags)) {
			window.allTags = allTags.tags;
		} else {
			window.allTags = [];
		}
		document.dispatchEvent(new CustomEvent("tags:loaded", { detail: { tags: window.allTags } }));
	} catch (error) {
		console.warn("Error initializing available tags:", error);
		window.allTags = [];
	}
});