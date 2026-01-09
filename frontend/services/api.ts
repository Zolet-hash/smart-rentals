export const api = {
  post: async (url: string, body: any) => {
    const res = await fetch(url, {
      method: "POST",
      headers: {
        "Content-Type": "application/json",
        Authorization: `Bearer ${localStorage.getItem("token") || ""}`,
      },
      body: JSON.stringify(body),
    });

    if (!res.ok) throw new Error("Request failed");
    return res.json();
  },

  get: async (url: string) => {
    const res = await fetch(url, {
      headers: {
        Authorization: `Bearer ${localStorage.getItem("token") || ""}`,
      },
    });

    if (!res.ok) throw new Error("Request failed");
    return res.json();
  },
};
