import http from "k6/http";
import { check, sleep } from "k6";
import { Counter, Trend, Rate } from "k6/metrics";

// Custom metrics
let loginSuccess = new Counter("login_success");
let loginFailures = new Counter("login_failures");
let sessionReadSuccess = new Counter("session_read_success");
let sessionReadFailures = new Counter("session_read_failures");
let logoutSuccess = new Counter("logout_success");
let logoutFailures = new Counter("logout_failures");
let sessionResponseTime = new Trend("session_response_time");
let loginResponseTime = new Trend("login_response_time");
let errorRate = new Rate("errors");

// Configuration knobs
const BASE_URL = __ENV.BASE_URL || "http://localhost:8080";
const PAYLOAD_SIZE = __ENV.PAYLOAD_SIZE || 1024;
const TEST_DURATION = __ENV.TEST_DURATION || "2m";
const RAMP_UP_TIME = __ENV.RAMP_UP_TIME || "30s";
const MAX_VUS = __ENV.MAX_VUS || 100;
const SESSION_TTL = __ENV.SESSION_TTL || 10;
const READ_WRITE_RATIO = __ENV.READ_WRITE_RATIO || 4; // 4:1 read:write ratio

export let options = {
  stages: [
    { duration: "30s", target: 500 }, // ramp-up
    { duration: "2m", target: 500 }, // main test
    { duration: "30s", target: 0 }, // ramp-down
  ],
  thresholds: {
    errors: ["rate<0.01"],
    http_req_duration: ["p(95)<500"],
    session_read_success: ["count>1000"],
    login_success: ["count>200"],
  },
};

export default function () {
  // 1. Login to create a new session
  let loginRes = http.post(`${BASE_URL}/login`);
  let loginOk = check(loginRes, {
    "login status is 200": (r) => r.status === 200,
    "login returned session ID": (r) => r.body.length > 0,
  });

  if (loginOk) {
    loginSuccess.add(1);
    loginResponseTime.add(loginRes.timings.duration);
  } else {
    loginFailures.add(1);
    errorRate.add(1);
  }

  let sessionId = loginRes.body;

  // Random sleep to simulate real user behavior pattern
  sleep(Math.random() * 2);

  // 2. Perform session reads
  let reads = Math.floor(Math.random() * READ_WRITE_RATIO) + 1;
  const BASE64_SIZE = Math.ceil(PAYLOAD_SIZE / 3) * 4;
  for (let i = 0; i < reads; i++) {
    let readRes = http.get(`${BASE_URL}/session/${sessionId}`);
    let readOk = check(readRes, {
      "session read status is 200": (r) => r.status === 200,
      "session data correct size": (r) => r.body.length === BASE64_SIZE,
    });

    if (readOk) {
      sessionReadSuccess.add(1);
      sessionResponseTime.add(readRes.timings.duration);
    } else {
      sessionReadFailures.add(1);
      errorRate.add(1);
    }

    // Random sleep between reads
    sleep(Math.random() * 1);
  }

  // 3. Logout
  if (Math.random() > 0.3) {
    // 70% of users logout
    let logoutRes = http.get(`${BASE_URL}/logout/${sessionId}`);
    let logoutOk = check(logoutRes, {
      "logout status is 200": (r) => r.status === 200,
    });

    if (logoutOk) {
      logoutSuccess.add(1);
    } else {
      logoutFailures.add(1);
      errorRate.add(1);
    }
  }
}
