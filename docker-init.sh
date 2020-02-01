cd /go/src/github.com/paulhatch/konfigraf
go get -d ./...
plgo .

# Build the extension files
cd build
make install with_llvm=no

# Get the name of the extension .sql file
SQL_FILE=$(ls /usr/share/postgresql/12/extension/*.sql | head -1)
# Lowercase the function names
sed -i -r 's/(CREATE OR REPLACE FUNCTION)([^\(]+)/\1\L\2/g' $SQL_FILE
# Copy our pre-written extension SQL into the generated extension file
cat ../extension.sql >> $SQL_FILE

# Export the extension artifact files
cp -r /usr/share/postgresql/12/extension /dist
cp -r /usr/lib/postgresql/12/lib /dist
