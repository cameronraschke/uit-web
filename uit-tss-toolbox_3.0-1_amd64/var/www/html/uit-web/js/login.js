let submitInProgress = false;

const loginForm = document.querySelector("#login-form");
const usernameInput = document.getElementById("username");
const passwordInput = document.getElementById("password");
const loginButton = document.getElementById("login-button");
const usernameStar = document.getElementById("username-star");
const passwordStar = document.getElementById("password-star");

function checkUsernameValidity() {
    const usernameValid = usernameInput.checkValidity();
    if (!usernameValid) {
        usernameStar.style.display = "block";
        usernameStar.style.color = "red";
    } else {
        usernameStar.style.display = "none";
        usernameStar.style.color = "black";
    }
}

function checkPasswordValidity() {
    const passwordValid = passwordInput.checkValidity();
    if (!passwordValid) {
        passwordStar.style.display = "block";
        passwordStar.style.color = "red";
    } else {
        passwordStar.style.display = "none";
        passwordStar.style.color = "black";
    }
}

checkUsernameValidity();
usernameInput.addEventListener("keyup", () => {
    checkUsernameValidity();
});

checkPasswordValidity();
passwordInput.addEventListener("keyup", () => {
    checkPasswordValidity();
});

loginForm.addEventListener("submit", async (event) => {
    if (submitInProgress) return;
    submitInProgress = true;
    event.preventDefault();
    const usernameValid = usernameInput.reportValidity();
    const passwordValid = passwordInput.reportValidity();
    const formData = new FormData(loginForm);
    if (!formData.has("username") || !formData.has("password")) {
        console.log("Username or password not provided");
        return;
    }
    if (formData.get("username").trim() === "" || formData.get("password").trim() === "") {
        console.log("Username or password is empty");
        return;
    }
    if (formData.get("username").length > 20 || formData.get("password").length > 64) {
        console.log("Username or password is too long");
        return;
    }
    if (formData.get("username").length < 3 || formData.get("password").length < 8) {
        console.log("Username or password is too short");
        return;
    }
    if (/\s/.test(formData.get("username")) || /\s/.test(formData.get("password"))) {
        console.log("Username or password contains whitespace");
        return;
    }
    if (!usernameValid || !passwordValid) {
        console.log("Invalid formatting in username or password\nUsername: " + usernameValid.validationMessage + "\n" + passwordValid.validationMessage);
        return;
    }

    const usernameValue = formData.get("username").trim();
    const passwordValue = formData.get("password").trim();

    try {
        const jsonArr = {
            username: await generateSHA256Hash(usernameValue),
            password: await generateSHA256Hash(passwordValue)
        };


        jsonData = JSON.stringify(jsonArr);
        if (!jsonData || jsonData.length === 0) throw new Error('No data to send to login API');

        const base64Payload = jsonToBase64(jsonData);
        if (!base64Payload || base64Payload.length === 0) throw new Error('Failed to encode login payload to base64');

        const response = await fetch('/login.html', {
            method: 'POST',
            headers: { 'Content-Type': 'application/json',
              'Content-Transfer-Encoding': 'base64'
            },
            credentials: "include",
            body: base64Payload
        });

        if (!response.ok) throw new Error('Network response was not ok: ' + response.statusText);
        const data = await response.json();
        if (!data || (typeof data === "object" && Object.keys(data).length === 0) || !data.token || data.token.length === 0) {
            throw new Error('No data returned from login API');
        }

        await setKeyFromIndexDB("bearerToken", data.token);
        window.location.href = "/index.html";
    } catch (error) {
        console.error('There was a problem with the fetch operation:', error);
        submitInProgress = false;
    } finally {
        submitInProgress = false;
    }
});

async function setKeyFromIndexDB(key, value) {
    try {
        if (!key || key.length === 0 || typeof key !== "string" || key.trim() === "") {
        throw new Error("Key is invalid: " + key);
        }
        if (!value || value.length === 0 || typeof value !== "string" || value.trim() === "") {
        throw new Error("Value is invalid: " + value);
        }

        await new Promise((resolve, reject) => {
            const tokenDBConnection = indexedDB.open("uitTokens", 1);
            tokenDBConnection.onsuccess = (event) => {
                const db = event.target.result;
                const transaction = db.transaction(["uitTokens"], "readwrite");
                const objectStore = transaction.objectStore("uitTokens");
                objectStore.put({ tokenType: key, value: value });
                transaction.oncomplete = () => resolve();
                transaction.onerror = (event) => reject("Error storing " + key + " in IndexedDB: " + event.target.error);
            };
            tokenDBConnection.onerror = (event) => reject("Error opening IndexedDB: " + event.target.error);
        });
        
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