import http from "k6/http";
import { check, sleep } from "k6";

export const options = {
  stages: [
    { duration: "30s", target: 500 }, // Ramp-up
    { duration: "2m", target: 500 }, // Peak load
    { duration: "30s", target: 0 }, // Ramp-down
  ],
  thresholds: {
    http_req_failed: ["rate<0.01"], // <1% errors
    http_req_duration: ["p(95)<500"], // 95% of requests <500ms
  },
};

export default function () {
  // --- PHASE 1: POST RANDOM SCORES ---
  const userId = Math.floor(Math.random() * 100000); // this will create a userId between 0 and 99999
  const score = Math.floor(Math.random() * 1000);

  const postRes = http.post(
    `http://localhost:8080/score/${userId}?score=${score}`,
    null,
    { timeout: "5s" }
  );

  check(postRes, {
    "POST /score succeeded": (r) => r.status === 200,
  });

  //PHASE 2: VALIDATE LEADERBOARD  30% of requests check leaderboard
  if (Math.random() < 0.3) {
    // 30% chance to check leaderboard
    const topN = 10;
    const getRes = http.get(`http://localhost:8080/leaderboard/top?n=${topN}`);

    check(getRes, {
      "GET /leaderboard succeeded": (r) => r.status === 200,
      "Leaderboard is correctly sorted": (r) => {
        const data = JSON.parse(r.body);
        for (let i = 1; i < data.length; i++) {
          if (data[i].score > data[i - 1].score) {
            console.error(
              `Sorting error: ${data[i].score} > ${data[i - 1].score}`
            );
            return false;
          }
        }
        return true;
      },
    });
  }

  sleep(0.5); // Simulate user response time
}
