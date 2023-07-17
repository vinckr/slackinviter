async function getSession() {
  const url = "https://auth.slackinviter.vinckr.com/sessions/whoami"; // Replace with the actual URL

  try {
    const response = await fetch(url, {
      credentials: "include",
      mode: "cors",
    });
    if (response.ok) {
      console.log("response: ", response); // Response object
      const responseData = await response.json();
      createAndSubmitForm(responseData);
    } else {
      console.error("Failed to fetch Ory Network session: ", response.status);
    }
  } catch (error) {
    console.error(error.message); // Error message
  }
}

function createAndSubmitForm(data) {
  const form = document.createElement("form");
  form.method = "POST";
  form.action = "/sessiondata";
  form.style.display = "none";

  const input = document.createElement("input");
  input.type = "hidden";
  input.name = "sessionData";
  input.value = JSON.stringify(data);

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
        console.error("Failed to fetch HTML content: ", response.status);
      }
    } catch (error) {
      console.error("Error while fetching HTML content: ", error.message);
    }
  });
  form.submit();
}

getSession();
