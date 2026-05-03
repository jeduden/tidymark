---
diagnostics:
  - line: 3
    column: 1
    message: 'build directive references unknown recipe "nonexistent"'
---
# Unknown Recipe

<?build
recipe: nonexistent
output: out.png
?>
content
<?/build?>
