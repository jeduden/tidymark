---
diagnostics:
  - line: 3
    column: 1
    message: 'build directive recipe "screenshot": missing required parameter "url"'
---
# Missing Required Param

<?build
recipe: screenshot
output: out.png
?>
content
<?/build?>
