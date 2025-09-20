const tokenDB = indexedDB.open("uitTokens", 1);
tokenDB.onupgradeneeded = (event) => {
  const db = event.target.result;
  console.log(`Upgrading to version ${db.version}`);

  const objectStore = db.createObjectStore("uitTokens", {
    keyPath: "tokenType",
  });

  objectStore.createIndex("authStr", "authStr", { unique: true });
  objectStore.createIndex("basicToken", "basicToken", { unique: true });
  objectStore.createIndex("bearerToken", "bearerToken", { unique: true });
};
// const tokenWorker = new Worker('js/auth-webworker.js');

function jsonToBase64(jsonString) {
    try {
        if (typeof jsonString !== 'string') {
            throw new TypeError("Input is not a valid JSON string");
        }

        const jsonParsed = JSON.parse(jsonString);
        if (!jsonParsed) {
            throw new TypeError("Input is not a valid JSON string");
        }
        if (jsonParsed && typeof jsonParsed === 'object' && Object.prototype.hasOwnProperty.call(jsonParsed, '__proto__')) {
            throw new Error(`Prototype pollution detected`);
        }

        const uft8Bytes = new TextEncoder().encode(jsonString);
        const base64JsonData = uft8Bytes.toBase64({ alphabet: "base64url" })
        // Decode json with base64ToJson and double-check that it's correct.
        const decodedJson = base64ToJson(base64JsonData);
        if (!base64JsonData || JSON.stringify(jsonParsed) !== decodedJson) {
            throw new Error(`Encoded json does not match decoded json. \n${base64JsonData}\n${decodedJson}`)
        }
        return base64JsonData;
    } catch (error) {
        console.error("Invalid JSON string:", error);
        return null;
    }
}

function base64ToJson(base64String) {
    try {
        if (typeof base64String !== 'string') {
            throw new TypeError("Input is not a valid base64 string");
        }
        if (base64String.trim() === "") {
            throw new Error("Base64 string is empty");
        }

        const base64Bytes = atob(base64String);
        const byteArray = new Uint8Array(base64Bytes.length);
        const decodeResult = byteArray.setFromBase64(base64String, { alphabet: "base64url" });
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
    }
}

async function fetchData(url, returnText = false, fetchOptions = {}) {
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
      ...(fetchOptions.signal ? { signal: fetchOptions.signal } : {})
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

async function getKeyFromIndexDB(key = null) {
    if (!key || key.length === 0 || typeof key !== "string" || key.trim() === "") {
      throw new Error("Key is invalid: " + key);
    }

    try {
        const dbConn = await new Promise((resolve, reject) => {
            const tokenDBConnection = indexedDB.open("uitTokens", 1);
            tokenDBConnection.onsuccess = (event) => resolve(event.target.result);
            tokenDBConnection.onerror = (event) => reject("Error opening IndexedDB: " + event.target.error);
        });

        const tokenTransaction = dbConn.transaction(["uitTokens"], "readwrite");
        const tokenObjectStore = tokenTransaction.objectStore("uitTokens");

        const tokenObj = await new Promise((resolve, reject) => {
            const tokenRequest = tokenObjectStore.get(key);
            tokenRequest.onsuccess = event => resolve(event.target.result);
            tokenRequest.onerror = event => reject("Error querying token from IndexedDB: " + event.target.error);
        });

        if (!tokenObj || !tokenObj.value || typeof tokenObj.value !== "string" || tokenObj.value.trim() === "") {
            throw new Error("No token found for key: " + key);
        }
        return tokenObj.value;
    } catch (error) {
        throw new Error("Error accessing IndexedDB: " + error);
    }
}

async function generateSHA256Hash(input) {
    if (!input || input.length === 0 || typeof input !== "string" || input.trim() === "") {
      throw new Error("Hash input is invalid: " + input);
    }

    const encoder = new TextEncoder();
    const encodedInput = encoder.encode(input);
    const hashBuffer = await crypto.subtle.digest("SHA-256", encodedInput);
    const hashArray = Array.from(new Uint8Array(hashBuffer));
    const hashStr = hashArray.map(b => b.toString(16).padStart(2, "0")).join("");
    if (!hashStr || hashStr.length === 0 || typeof hashStr !== "string" || hashStr.trim() === "") {
      throw new Error("Hash generation failed: " + input);
    }
    return hashStr;
}
