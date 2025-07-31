import json
import matplotlib.pyplot as plt
import numpy as np

# Load the k6 summary data from the provided JSON file content.
# In a real script, you would load this from a file:
# with open('summary.json', 'r') as f:
#     data = json.load(f)
data = {
    "root_group": {
        "id": "d41d8cd98f00b204e9800998ecf8427e",
        "groups": {},
        "checks": {
                "login status is 200": {
                    "id": "baba87e761865dc93085ee0c1fe25f4c",
                    "passes": 2712,
                    "fails": 0,
                    "name": "login status is 200",
                    "path": "::login status is 200"
                }
            },
        "name": "",
        "path": ""
    },
    "metrics": {
        "http_req_duration": {
            "avg": 0.9606596011565738,
            "min": 0.057301,
            "med": 0.403969,
            "max": 71.720434,
            "p(90)": 0.6380496,
            "p(95)": 0.7012721999999998,
            "thresholds": {
                "p(95)<500": False
            }
        },
        "http_reqs": {
            "count": 11413,
            "rate": 179.71526784753712
        },
        "http_req_sending": {
            "min": 0.002775,
            "med": 0.01923,
            "max": 2.188561,
            "p(90)": 0.0322676,
            "p(95)": 0.036799599999999995,
            "avg": 0.023478309121177626
        },
        "http_req_waiting": {
            "p(95)": 0.5879862,
            "avg": 0.8835414769999187,
            "min": 0.044538,
            "med": 0.325867,
            "max": 71.479633,
            "p(90)": 0.5304636
        },
        "http_req_receiving": {
            "avg": 0.05363981503548595,
            "min": 0.006205,
            "med": 0.056291,
            "max": 1.537596,
            "p(90)": 0.08205960000000001,
            "p(95)": 0.0911984
        }
    }
}


# --- Function to create all plots ---
def create_visualizations(metrics_data):
    """
    Generates and displays visualizations for k6 test metrics.
    - Latency Histogram
    - Throughput Bar Chart
    - Request Timing Breakdown Stacked Bar Chart
    """
    
    # --- 1. Latency Visualization (Histogram) ---
    print("Generating Latency Histogram...")
    http_req_duration = metrics_data['http_req_duration']
    http_reqs_count = metrics_data['http_reqs']['count']
    
    # Since the summary only has statistics, we generate random data
    # that approximates the given distribution for visualization purposes.
    # For a perfectly accurate histogram, you would need raw k6 output.
    avg = http_req_duration['avg']
    med = http_req_duration['med']
    p95 = http_req_duration['p(95)']
    # Approximate standard deviation from the p(95) value
    std_dev = (p95 - avg) / 1.645  
    np.random.seed(42) # for reproducible results
    
    # Generate a log-normal distribution which often models response times well
    latency_data = np.random.lognormal(mean=np.log(med), sigma=std_dev / avg, size=int(http_reqs_count))
    latency_data[latency_data < 0] = http_req_duration['min'] # Ensure no negative values

    plt.style.use('seaborn-v0_8-whitegrid')
    fig, (ax1, ax2, ax3) = plt.subplots(3, 1, figsize=(10, 18))
    fig.suptitle('k6 Load Test Performance Analysis', fontsize=20, y=0.95)
    
    ax1.hist(latency_data, bins=50, alpha=0.75, color='cornflowerblue', edgecolor='black', range=(0, p95 * 2))
    ax1.axvline(avg, color='red', linestyle='dashed', linewidth=2, label=f'Average: {avg:.2f} ms')
    ax1.axvline(med, color='green', linestyle='dashed', linewidth=2, label=f'Median: {med:.2f} ms')
    ax1.axvline(p95, color='purple', linestyle='dashed', linewidth=2, label=f'p(95): {p95:.2f} ms')
    
    ax1.set_title('HTTP Request Duration (Latency) Distribution', fontsize=16)
    ax1.set_xlabel('Request Duration (ms)', fontsize=12)
    ax1.set_ylabel('Number of Requests', fontsize=12)
    ax1.legend()

    # --- 2. Throughput Visualization (Bar Chart) ---
    print("Generating Throughput Bar Chart...")
    throughput_rate = metrics_data['http_reqs']['rate']
    
    ax2.bar(['Throughput'], [throughput_rate], color='lightcoral', width=0.4, zorder=3)
    ax2.set_title('Average Throughput', fontsize=16)
    ax2.set_ylabel('Requests per Second (reqs/s)', fontsize=12)
    ax2.text(0, throughput_rate, f'{throughput_rate:.2f}', ha='center', va='bottom', fontsize=12, fontweight='bold')
    ax2.set_ylim(0, throughput_rate * 1.25) # Set y-axis limit for better spacing

    # --- 3. Request Timing Breakdown (Stacked Bar Chart) ---
    print("Generating Request Timing Breakdown...")
    sending_avg = metrics_data['http_req_sending']['avg']
    waiting_avg = metrics_data['http_req_waiting']['avg']
    receiving_avg = metrics_data['http_req_receiving']['avg']
    
    labels = ['Average Request']
    timings = {
        'Sending': [sending_avg],
        'Waiting (TTFB)': [waiting_avg],
        'Receiving': [receiving_avg],
    }

    bottom = np.zeros(len(labels))
    for name, timing_data in timings.items():
        p = ax3.bar(labels, timing_data, width=0.5, label=f'{name} ({timing_data[0]:.2f} ms)', bottom=bottom, zorder=3)
        bottom += timing_data

    ax3.set_title('Average HTTP Request Duration Breakdown', fontsize=16)
    ax3.set_ylabel('Time (ms)', fontsize=12)
    ax3.legend(title='Request Phase')
    
    # Add total time text
    total_time = sending_avg + waiting_avg + receiving_avg
    ax3.text(0, total_time, f'Total: {total_time:.2f} ms', ha='center', va='bottom', fontsize=12, fontweight='bold')
    ax3.set_ylim(0, total_time * 1.25)

    plt.tight_layout(rect=[0, 0, 1, 0.93])
    plt.show()

# --- Main execution ---
if __name__ == "__main__":
    # Extract the metrics object from the full summary
    metrics = data['metrics']
    create_visualizations(metrics)

