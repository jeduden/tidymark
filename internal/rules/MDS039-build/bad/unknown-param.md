---
diagnostics:
  - line: 3
    column: 1
    message: 'build directive recipe "vhs": unknown parameter "bogus"'
---
# Unknown Param

<?build
recipe: vhs
input: demo.tape
output: demo.gif
bogus: value
?>
![vhs output: demo.gif](demo.gif)
<?/build?>
