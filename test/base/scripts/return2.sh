
echo hello11

AAA=$(return 67 2>&1)
echo "<$AAA=$?>"
echo hello22
return 123