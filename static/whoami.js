function replaceHTMLContent(htmlContent) {
  console.log("htmlContent: ", htmlContent);
  document.body.innerHTML = htmlContent;
}

async function getSession() {
  const url = "https://auth.slackinviter.vinckr.com/sessions/whoami"; // Replace with the actual URL

  try {
    const response = await fetch(url, {
      credentials: "include",
      mode: "cors",
    });
    console.log("response: ", response); // Response object
    const data = await response.json();
    const htmlContent = await response.text();
    replaceHTMLContent(htmlContent);
    console.log("session data: ", data); // Session object
    fetch("/sessiondata", {
      method: "POST",
      headers: {
        "Content-Type": "application/json",
      },
      body: JSON.stringify(data),
    });
  } catch (error) {
    console.error(error.message); // Error message
  }
}

getSession();
