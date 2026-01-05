interface Window {
  availableTags: string[];
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
		const base64JsonData = btoa(binaryStr).replace(/\+/g, "-").replace(/\//g, "_").replace(/=+$/, "");
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

async function fetchData(url: string, returnText = false, fetchOptions: RequestInit = {}): Promise<any> {
  try {
    if (!url || url.trim().length === 0) {
      throw new Error("No URL specified for fetchData");
    }

    // Get bearerToken from IndexedDB
    const bearerToken = await getKeyFromIndexDB("bearerToken");
    const headers = new Headers();
    headers.append('Content-Type', 'application/x-www-form-urlencoded');
    headers.append('credentials', 'same-origin');
    headers.append('Authorization', 'Bearer ' + bearerToken);
    

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

function openTokenDB() {
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

async function getAllTags(fetchOptions: RequestInit = {}) {
  const url = "/api/all_tags";
  try {
    const data = await fetchData(url, true, fetchOptions);
    if (!data) {
      console.warn("No data returned from /api/all_tags");
      return [];
    }

    let tagArr: string[] = [];
    if (!Array.isArray(data) && typeof data === "string") {
      try {
        tagArr = data
          .replace(/\[/, "")
          .replace(/\]/, "")
          .split(",")
          .map(tag => tag.trim())
          .filter(Boolean);
      } catch (error) {
        console.warn("/api/all_tags cannot be parsed as JSON: " + error);
        return [];
      }
    }

    if (!Array.isArray(tagArr)) {
      console.warn("/api/all_tags did not return an array");
      return [];
    }
    tagArr = tagArr.map(tag => (typeof tag === "number" ? String(tag) : String(tag || "").trim())).filter(tag => tag.length === 6);
    return tagArr;
  } catch (error) {
    console.error("Error fetching tags from /api/all_tags:", error);
    return [];
  }
}

document.addEventListener("DOMContentLoaded", () => {
  getAllTags()
    .then(tags => {
      window.availableTags = Array.isArray(tags) ? tags : [];
      document.dispatchEvent(new CustomEvent("tags:loaded", { detail: { tags: window.availableTags } }));
    })
    .catch(error => {
      console.warn("Error initializing available tags:", error);
      window.availableTags = [];
    });
});

async function waitForNextPaint(frames = 1) {
  while (frames-- > 0) {
    await new Promise(requestAnimationFrame);
  }
}