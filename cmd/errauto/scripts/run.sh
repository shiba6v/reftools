go install .
FILE=example/input/1/main.go
SIZE=$(cat ${FILE} | wc -c | tr -d " ")
echo $FILE"\n"$SIZE | cat - - example/input/1/main.go | errauto -file ${FILE} -offset 10
# わからねえ。おとなしくVSCodeから使うか。