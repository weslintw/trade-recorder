<script>
  import { onMount, afterUpdate } from 'svelte';
  import { Chart, registerables } from 'chart.js';

  export let data = [];

  let chartCanvas;
  let chartInstance;

  Chart.register(...registerables);

  onMount(() => {
    createChart();
  });

  afterUpdate(() => {
    if (chartInstance) {
      updateChart();
    } else {
      createChart();
    }
  });

  function createChart() {
    if (!chartCanvas || data.length === 0) return;

    const ctx = chartCanvas.getContext('2d');

    chartInstance = new Chart(ctx, {
      type: 'line',
      data: {
        labels: data.map(d => d.date),
        datasets: [{
          label: '累積盈虧',
          data: data.map(d => d.equity),
          borderColor: '#667eea',
          backgroundColor: 'rgba(102, 126, 234, 0.1)',
          borderWidth: 3,
          fill: true,
          tension: 0.4,
          pointRadius: 4,
          pointHoverRadius: 6,
          pointBackgroundColor: '#667eea',
          pointBorderColor: '#fff',
          pointBorderWidth: 2
        }]
      },
      options: {
        responsive: true,
        maintainAspectRatio: false,
        plugins: {
          legend: {
            display: true,
            position: 'top',
            labels: {
              font: {
                size: 14,
                weight: 'bold'
              },
              color: '#2d3748'
            }
          },
          tooltip: {
            mode: 'index',
            intersect: false,
            backgroundColor: 'rgba(0, 0, 0, 0.8)',
            titleColor: '#fff',
            bodyColor: '#fff',
            borderColor: '#667eea',
            borderWidth: 1,
            padding: 12,
            displayColors: true,
            callbacks: {
              label: function(context) {
                let label = context.dataset.label || '';
                if (label) {
                  label += ': ';
                }
                label += context.parsed.y.toFixed(2);
                return label;
              }
            }
          }
        },
        scales: {
          x: {
            grid: {
              display: false
            },
            ticks: {
              color: '#718096',
              font: {
                size: 12
              }
            }
          },
          y: {
            grid: {
              color: 'rgba(0, 0, 0, 0.05)'
            },
            ticks: {
              color: '#718096',
              font: {
                size: 12
              },
              callback: function(value) {
                return value.toFixed(0);
              }
            }
          }
        },
        interaction: {
          mode: 'nearest',
          axis: 'x',
          intersect: false
        }
      }
    });
  }

  function updateChart() {
    if (!chartInstance || data.length === 0) return;

    chartInstance.data.labels = data.map(d => d.date);
    chartInstance.data.datasets[0].data = data.map(d => d.equity);
    chartInstance.update();
  }

  function destroyChart() {
    if (chartInstance) {
      chartInstance.destroy();
      chartInstance = null;
    }
  }

  // 當元件被銷毀時清理圖表
  import { onDestroy } from 'svelte';
  onDestroy(() => {
    destroyChart();
  });
</script>

<div class="chart-wrapper">
  <canvas bind:this={chartCanvas}></canvas>
</div>

<style>
  .chart-wrapper {
    position: relative;
    height: 400px;
    width: 100%;
  }

  canvas {
    max-width: 100%;
  }
</style>

