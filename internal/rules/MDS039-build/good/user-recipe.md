---
settings:
  recipes:
    chart:
      body-template: "[{output}]({output})"
      params:
        required:
          - data
---
# Chart Build

<?build
recipe: chart
data: data.csv
output: chart.png
?>
[chart.png](chart.png)
<?/build?>
