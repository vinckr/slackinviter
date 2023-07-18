async function whoami() {
  try {
    const sessionData = await getSession();
    console.log("session data: ", sessionData);
    createAndSubmitForm(sessionData);
  } catch (error) {
    console.error(
      "Error during session retrieval and form submission: ",
      error
    );
  }
}
async function getSession() {
  const url = "https://auth.slackinviter.vinckr.com/sessions/whoami"; // Replace with the actual URL
  console.log("getting session data");
  try {
    const response = await fetch(url, {
      credentials: "include",
      mode: "cors",
    });
    /*     const response = {
      id: "d4f5bb7e-d937-4d87-a0b7-0927312cdebd",
      active: true,
      expires_at: "2023-07-20T12:32:17.409035Z",
      identity: {
        id: "eb813b51-3e69-4b72-ae91-7fa2303aa39b",
        state: "active",
        traits: { email: "vincent+test123@ory.sh", name: "asd" },
      },
    }; */

    console.log("response: ", response);

    if (response.ok) {
      const responseData = await response.json();
      return responseData;
    } else {
      throw new Error(
        "Failed to fetch Ory Network session: " + response.status
      );
    }
  } catch (error) {
    console.error(error.message); // Error message
    throw error; // Rethrow the error to propagate it to the caller
  }
}
async function createAndSubmitForm(sessionData) {
  const form = document.createElement("form");
  form.method = "POST";
  form.action = "/invitation";
  form.style.display = "none";
  const input = document.createElement("input");
  input.type = "hidden";
  input.name = "sessionData";
  input.value = JSON.stringify(sessionData);
  form.appendChild(input);
  document.body.appendChild(form);
  form.addEventListener("submit", async (event) => {
    event.preventDefault();
    const formData = new FormData(form);
    try {
      const response = await fetch(form.action, {
        method: form.method,
        body: formData,
      });
      if (response.ok) {
        const htmlContent = await response.text();
        document.body.innerHTML = htmlContent;
      } else {
        throw new Error("Failed to fetch HTML content: " + response.status);
      }
    } catch (error) {
      console.error("Error while fetching HTML content: ", error.message);
      throw error; // Rethrow the error to propagate it to the caller
    }
  });
  form.submit();
}
document.addEventListener("DOMContentLoaded", () => {
  // Execute when the document has finished loading
  whoami();
});
