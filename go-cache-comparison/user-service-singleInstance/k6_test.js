import http from "k6/http";
import { check, sleep } from "k6";

export const options = {
  stages: [
    { duration: "10s", target: 20 }, // Ramp-up
    { duration: "80s", target: 100 }, // Peak load
    { duration: "10s", target: 0 }, // Ramp-down
  ],
  thresholds: {
    http_req_failed: ["rate<0.01"], // <1% errors
    http_req_duration: ["p(95)<500"], // 95% of requests <500ms
  },
};

export default function () {
  // --- PHASE 1: Fetch User Details ---
  const userId = Math.floor(Math.random() * 10000) + 1; // Random user ID between 1 and 10000
  const userRes = http.get(`http://localhost:8080/user?id=${userId}`);

  check(userRes, {
    "GET /user succeeded": (r) => r.status === 200,
    "Response contains user data": (r) =>
      r.body.includes("name") && r.body.includes("email"),
  });

  sleep(0.5); // Simulate user think time
}
