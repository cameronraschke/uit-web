let isRefreshingToken = false;
let isRequestingNewToken = false;
let tokenRequestPromise = null;

async function generateSHA256Hash(text = null) {
  try {
    if (!text) {
      throw new Error('Cannot hash empty string');
    }
    const encoder = new TextEncoder();
    const buffer = encoder.encode(text);
    const hashBuffer = await crypto.subtle.digest('SHA-256', buffer);
    const hash = Array.from(new Uint8Array(hashBuffer)).map(b => b.toString(16).padStart(2, '0')).join('');
    if (!hash) throw new Error('Hashing returned empty string');
    return hash;
  } catch (error) {
    console.error("Cannot hash string: " + error);
    return null;
  }
}

function openTokenDB() {
  return new Promise((resolve, reject) => {
    const request = indexedDB.open("uitTokens", 1);
    request.onupgradeneeded = (event) => {
      const db = event.target.result;
      const objectStore = db.createObjectStore("uitTokens", { keyPath: "tokenType" });
      objectStore.createIndex("authStr", "authStr", { unique: true });
      objectStore.createIndex("basicToken", "basicToken", { unique: true });
      objectStore.createIndex("bearerToken", "bearerToken", { unique: true });
    };
    request.onsuccess = event => resolve(event.target.result);
    request.onerror = event => reject("Cannot open token DB: " + event.target.error);
  });
}

function getTokenObject(db, key) {
  return new Promise((resolve, reject) => {
    const tx = db.transaction(["uitTokens"], "readonly");
    const store = tx.objectStore("uitTokens");
    const req = store.get(key);
    req.onsuccess = event => resolve(event.target.result);
    req.onerror = event => reject("Error retrieving " + key + " from IndexedDB: " + event.target.error);
  });
}

function putTokenObject(db, key, value) {
  return new Promise((resolve, reject) => {
    const tx = db.transaction(["uitTokens"], "readwrite");
    const store = tx.objectStore("uitTokens");
    const req = store.put({ tokenType: key, value });
    req.onsuccess = () => resolve();
    req.onerror = event => reject("Error storing " + key + " in IndexedDB: " + event.target.error);
  });
}

async function checkAndUpdateTokenDB() {
  if (isRequestingNewToken) return;
  isRequestingNewToken = true;
  let db = undefined;
  try {
    db = await openTokenDB();
    const authStrObj = await getTokenObject(db, "authStr");
    if (!authStrObj || !authStrObj.value) throw new Error('authStr is null or empty');
    const basicToken = await generateSHA256Hash(authStrObj.value);
    if (!basicToken) throw new Error('basicToken hashing failed');
    await putTokenObject(db, "basicToken", basicToken);

    // Check if bearerToken exists and is valid, otherwise request a new one
    const bearerTokenObj = await getTokenObject(db, "bearerToken");
    let bearerToken;
    if (bearerTokenObj && bearerTokenObj.value) {
      bearerToken = bearerTokenObj.value;
    } else {
      bearerToken = null;
    }

    let result;
    if (bearerToken) {
      result = await checkToken(bearerToken);
    } else {
      result = { valid: false, ttl: 0 };
    }

    if (!result.valid || Number(result.ttl) <= 5) {
      console.log("Bearer token is invalid or expired, requesting new token from API.");
      const newBearerToken = await getOrRequestToken();
      if (!newBearerToken) {
        throw new Error('Failed to retrieve new bearerToken');
      }
      const newResult = await checkToken(newBearerToken);
      if (!newResult.valid) {
        throw new Error('New bearerToken is invalid after retrieval');
      }
    }

    db.close();
    return;
  } catch (error) {
    if (db) db.close();
    isRequestingNewToken = false;
    throw error;
  } finally {
    isRequestingNewToken = false;
  }
}

async function checkToken(bearerToken = null) {
  return new Promise((resolve, reject) => {
    if (!bearerToken) {
      reject("No bearerToken provided to checkToken function");
      return;
    }

    const headers = new Headers({
      'Content-Type': 'application/x-www-form-urlencoded',
      'credentials': 'include',
      'Authorization': 'Bearer ' + bearerToken
    });

    const requestOptions = {
      method: 'GET',
      headers: headers
    };

    fetch('https://UIT_WAN_IP_ADDRESS:31411/api/auth?type=check-token', requestOptions)
      .then(response => {
        if (!response.ok) {
          console.log("Web server error while checking token: " + response.statusText);
          resolve({ valid: false, ttl: 0 });
          return;
        }
        return response.json();
      })
      .then(data => {
        if (!data || (typeof data === "object" && Object.keys(data).length === 0)) {
          console.log('No data returned from token check API');
          resolve({ valid: false, ttl: 0 });
          return;
        }

        if (data.token && Number(data.ttl) >= 5 && (data.valid === true || data.valid === "true")) {
          resolve({ valid: true, ttl: Number(data.ttl) });
          return;
        } else {
          resolve({ valid: false, ttl: 0 });
          return;
        }
      })
      .catch(error => {
        console.error("Error checking token validity: " + error);
        resolve({ valid: false, ttl: 0 });
      });
  });
}

async function newToken() {
  return new Promise((resolve, reject) => {
    // If token is invalid or expired, request a new one from the API
    const tokenDB = indexedDB.open("uitTokens", 1);
    tokenDB.onsuccess = function(event) {
      const db = event.target.result;
      const tokenTransaction = db.transaction(["uitTokens"], "readwrite");
      const tokenObjectStore = tokenTransaction.objectStore("uitTokens");

      // Get basicToken
      const basicTokenRequest = tokenObjectStore.get("basicToken")
      basicTokenRequest.onsuccess = function(event) {
        const basicTokenObj = event.target.result;
        if (!basicTokenObj || !basicTokenObj.value) {
          reject('basicToken object is invalid');
          return;
        }
        const basicToken = basicTokenObj.value;

        fetch('https://UIT_WAN_IP_ADDRESS:31411/api/auth?type=new-token', {
          method: 'GET',
          headers: {
            'Content-Type': 'application/x-www-form-urlencoded',
            'credentials': 'include',
            'Authorization': 'Basic ' + basicToken
          },
          body: null
        })
        .then(response => {
          if (!response.ok) {
            reject(`Response status: ${response.status}`);
            return;
          }
          return response.json();
        })
        .then(data => {
          if (data.token && (data.valid === true || data.valid === "true")) {
            const newTransaction = db.transaction(["uitTokens"], "readwrite");
            const newObjectStore = newTransaction.objectStore("uitTokens");
            const bearerTokenPutRequest = newObjectStore.put({ tokenType: "bearerToken", value: data.token });
            bearerTokenPutRequest.onsuccess = function() {
              resolve(data.token);
              return;
            };
            bearerTokenPutRequest.onerror = function(event) {
              reject("Error storing new bearerToken in IndexedDB: " + event.target.error);
              return;
            };
          } else {
            reject('No data returned from newToken API');
            return;
          }
        })
        .catch(error => {
          reject("Error fetching new bearerToken: " + error);
          return;
        });
      };
      basicTokenRequest.onerror = function(event) {
        reject("Error fetching basicToken: " + event.target.error);
        return;
      };
    };
    tokenDB.onerror = function(event) {
      reject("Cannot open token DB to create new token: " + event.target.error);
      return;
    };
  });
}

// Wait for a postMessage command from getCreds function
// self.addEventListener("message", async (event) => {
//   if (event.data.type === "updateTokenDB") {
//     checkAndUpdateTokenDB();
//   }
// });

// async function periodicTokenCheck() {
//   if (isRefreshingToken) {
//     return;
//   }
//   isRefreshingToken = true;
//   try {
//     await checkAndUpdateTokenDB();
//   } catch (error) {
//     console.error("Error in checkAndUpdateTokenDB:", error);
//   }
//   // Wait 1 second after checkAndUpdateTokenDB
//   isRefreshingToken = false;
//   setTimeout(periodicTokenCheck, 1000);
// }

function getOrRequestToken() {
  if (tokenRequestPromise) {
    // If a request is already in progress, return the same promise
    return tokenRequestPromise;
  }
  tokenRequestPromise = newToken()
    .then(token => {
      tokenRequestPromise = null;
      return token;
    })
    .catch(error => {
      tokenRequestPromise = null;
      throw error;
    });
  return tokenRequestPromise;
}

async function periodicTokenCheck() {
  while (true) {
    if (!isRefreshingToken) {
      isRefreshingToken = true;
      try {
        await checkAndUpdateTokenDB();
      } catch (error) {
        console.error("Error in checkAndUpdateTokenDB:", error);
      }
      isRefreshingToken = false;
    }
    await new Promise(resolve => setTimeout(resolve, 1000));
  }
}

periodicTokenCheck();
