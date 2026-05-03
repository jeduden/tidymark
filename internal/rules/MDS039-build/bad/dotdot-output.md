---
diagnostics:
  - line: 3
    column: 1
    message: 'build directive "output" contains ".." path component: "../out/file.png"'
---
# Dotdot Output

<?build
recipe: screenshot
url: /inbox
output: ../out/file.png
?>
content
<?/build?>
