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
        const payload = {
            username: usernameValue,
            password: passwordValue
        };

        jsonData = JSON.stringify(payload);
        if (!jsonData || jsonData.length === 0) throw new Error('No data to send to login API');

        const response = await fetch('/login.html', {
            method: 'POST',
            headers: { 'Content-Type': 'application/json'},
            credentials: "include",
            body: jsonData
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