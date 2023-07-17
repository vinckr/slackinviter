async function getSession() {
  const url = "https://auth.slackinviter.vinckr.com/sessions/whoami"; // Replace with the actual URL

  try {
    const response = await fetch(url, {
      credentials: "include",
      mode: "cors",
    });

    console.log("response: ", response); // Response object

    if (response.ok) {
      const responseData = await response.json();
      return responseData; // Resolve the promise with the response data
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

async function createAndSubmitForm(data) {
  const form = document.createElement("form");
  form.method = "POST";
  form.action = "/sessiondata";
  form.style.display = "none";

  const input = document.createElement("input");
  input.type = "hidden";
  input.name = "sessionData";
  input.value = data;

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

// getSession
getSession()
  .then(createAndSubmitForm)
  .catch((error) => {
    console.error(
      "Error during session retrieval and form submission: ",
      error
    );
  });
