package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

func main() {
	rk, dsk := []string{}, []string{}
	sr, sds := map[string]string{}, map[string]string{}
	globalKeysRes := map[string]interface{}{}
	globalKeysDS := map[string]interface{}{}
	err := filepath.Walk("stackit/internal/",
		func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}
			if info.IsDir() {
				return nil
			}
			if strings.HasPrefix(path, "stackit/internal/data-sources") && strings.HasSuffix(path, "_test.go") {
				sl := strings.Split(path, "/")
				key := strings.Join(sl[3:len(sl)-1], " ")
				if _, ok := sds[key]; ok {
					return nil
				}
				globalKeysRes[sl[3]] = nil
				dsk = append(dsk, key)
				sds[key] = strings.Join(sl[:len(sl)-1], "/")
			}
			if strings.HasPrefix(path, "stackit/internal/resources") && strings.HasSuffix(path, "_test.go") {
				sl := strings.Split(path, "/")
				key := strings.Join(sl[3:len(sl)-1], " ")
				if _, ok := sr[key]; ok {
					return nil
				}
				globalKeysDS[sl[3]] = nil
				rk = append(rk, key)
				sr[key] = strings.Join(sl[:len(sl)-1], "/")
			}
			return nil
		})
	if err != nil {
		log.Println(err)
	}

	sortedGlobalKeysRes := []string{}
	for g := range globalKeysRes {
		sortedGlobalKeysRes = append(sortedGlobalKeysRes, g)
	}

	sortedGlobalKeysDS := []string{}
	for g := range globalKeysDS {
		sortedGlobalKeysDS = append(sortedGlobalKeysDS, g)
	}

	sort.Strings(sortedGlobalKeysRes)
	sort.Strings(sortedGlobalKeysDS)
	sort.Strings(rk)
	sort.Strings(dsk)

	// fmt.Println("found resources:")
	// printOutcome(sortedGlobalKeysRes, rk, sr)

	// fmt.Println("\nfound data sources:")
	// printOutcome(sortedGlobalKeysDS, dsk, sds)

	s := "# Code generated by 'make pre-commit' DO NOT EDIT."
	data, err := ioutil.ReadFile(".github/files/generate-acceptance-tests/template.yaml")
	if err != nil {
		fmt.Println(err)
	}
	sData := string(data)

	dsstr := printDataSourceOutcome(sortedGlobalKeysDS, dsk, sds, "datasource-")
	resstr, deleteNeeds := printResourceOutcome(sortedGlobalKeysRes, rk, sr, "resource-", "datasource-")
	sData = strings.Replace(sData, "__data_sources__", dsstr, 1)
	sData = strings.Replace(sData, "__resources__", resstr, 1)
	sData = strings.Replace(sData, "__delete_needs__", deleteNeeds, 2)

	err = ioutil.WriteFile(".github/workflows/acceptance_test.yml", []byte(s+sData), 0644)
	if err != nil {
		fmt.Println(err)
	}
}

func printDataSourceOutcome(sortedglobalKeys []string, sortedKeys []string, keyAndPathMap map[string]string, prefix string) string {
	s := ""
	nextNeeds := []string{"datasources"}

	// sort keys and names with their prefixes
	sorted := map[string][]string{}
	for _, key := range sortedglobalKeys {
		if _, ok := sorted[key]; !ok {
			sorted[key] = []string{}
		}
		for _, name := range sortedKeys {
			if !strings.HasPrefix(name, key) {
				continue
			}
			sorted[key] = append(sorted[key], name)
		}
	}
	// handle restricted matrix
	for _, id := range sortedglobalKeys {
		names := sorted[id]
		if len(names) < 2 {
			continue
		}
		sort.Strings(names)
		nextNeeds = append(nextNeeds, prefix+id)
		incl := ""
		for _, n := range names {
			incl = incl + fmt.Sprintf(`
        - name: %s
          path: %s
`, n, keyAndPathMap[n])
		}
		s = s + fmt.Sprintf(`
  %s%s:
    strategy:
      fail-fast: false
      max-parallel: 1
      matrix:
        name: [%s]
        include:
%s
    name: ${{ matrix.name }} data source
    needs: createproject
    runs-on: ubuntu-latest
    if: always()
    steps:
      - name: Checkout
        uses: actions/checkout@v3
      - name: Set up Go
        uses: actions/setup-go@v3
        with:
          go-version-file: 'go.mod'
          check-latest: true
          cache: true
      - name: Test ${{ matrix.name }} Data Source
        run: |
          echo $path
          export ACC_TEST_PROJECT_ID=${{needs.createproject.outputs.projectID}}
          if [[ -z "${ACC_TEST_PROJECT_ID}" || "${ACC_TEST_PROJECT_ID}" == "NULL" || "${ACC_TEST_PROJECT_ID}" == "null" ]]; then
            exit 1;
          fi;
          make ci-testacc TEST="./${{ matrix.path }}/..." ACC_TEST_BILLING_REF="${{ secrets.ACC_TEST_BILLING_REF }}" ACC_TEST_USER_EMAIL="${{ secrets.ACC_TEST_USER_EMAIL }}" STACKIT_SERVICE_ACCOUNT_TOKEN="${{ secrets.STACKIT_SERVICE_ACCOUNT_TOKEN }}" STACKIT_SERVICE_ACCOUNT_EMAIL="${{ secrets.STACKIT_SERVICE_ACCOUNT_EMAIL }}"
      - name: Save results
        if: always()
        uses: actions/upload-artifact@v3
        with:
          path: .github/files/analyze-test-output/result/*.json
`, prefix, id, strings.Join(names, ","), incl)
	}

	// handle non restricted matrix
	// collect names
	collectedNames := []string{}
	for _, names := range sorted {
		if len(names) != 1 {
			continue
		}
		collectedNames = append(collectedNames, names...)
	}
	sort.Strings(collectedNames)
	incl := ""
	for _, n := range collectedNames {
		incl = incl + fmt.Sprintf(`
        - name: %s
          path: %s
`, n, keyAndPathMap[n])
	}

	s = s + fmt.Sprintf(`
  datasources:
    strategy:
      fail-fast: false
      matrix:
        name: [%s]
        include:
%s
    name: ${{ matrix.name }} data source
    needs: createproject
    runs-on: ubuntu-latest
    if: always()
    steps:
      - name: Checkout
        uses: actions/checkout@v3
      - name: Set up Go
        uses: actions/setup-go@v3
        with:
          go-version-file: 'go.mod'
          check-latest: true
          cache: true
      - name: Test ${{ matrix.name }} Data Source
        run: |
          export ACC_TEST_PROJECT_ID=${{needs.createproject.outputs.projectID}}
          if [[ -z "${ACC_TEST_PROJECT_ID}" || "${ACC_TEST_PROJECT_ID}" == "NULL" || "${ACC_TEST_PROJECT_ID}" == "null" ]]; then
            exit 1;
          fi;
          make ci-testacc TEST="./${{ matrix.path }}/..." ACC_TEST_BILLING_REF="${{ secrets.ACC_TEST_BILLING_REF }}" ACC_TEST_USER_EMAIL="${{ secrets.ACC_TEST_USER_EMAIL }}" STACKIT_SERVICE_ACCOUNT_TOKEN="${{ secrets.STACKIT_SERVICE_ACCOUNT_TOKEN }}" STACKIT_SERVICE_ACCOUNT_EMAIL="${{ secrets.STACKIT_SERVICE_ACCOUNT_EMAIL }}"
      - name: Save results
        if: always()
        uses: actions/upload-artifact@v3
        with:
          path: .github/files/analyze-test-output/result/*.json
`, strings.Join(collectedNames, ","), incl)

	return s

}

func printResourceOutcome(sortedglobalKeys []string, sortedKeys []string, keyAndPathMap map[string]string, prefix, previousPrefix string) (string, string) {
	s := ""
	nextNeeds := []string{"resources"}

	// sort keys and names with their prefixes
	sorted := map[string][]string{}
	for _, key := range sortedglobalKeys {
		if _, ok := sorted[key]; !ok {
			sorted[key] = []string{}
		}
		for _, name := range sortedKeys {
			if !strings.HasPrefix(name, key) {
				continue
			}
			sorted[key] = append(sorted[key], name)
		}
	}
	// handle restricted matrix
	for _, id := range sortedglobalKeys {
		names := sorted[id]
		if len(names) < 2 {
			continue
		}
		sort.Strings(names)
		nextNeeds = append(nextNeeds, prefix+id)
		incl := ""
		for _, n := range names {
			incl = incl + fmt.Sprintf(`
        - name: %s
          path: %s
`, n, keyAndPathMap[n])
		}
		s = s + fmt.Sprintf(`
  %s%s:
    strategy:
      fail-fast: false
      max-parallel: 1
      matrix:
        name: [%s]
        include:
%s
    name: ${{ matrix.name }} resource
    needs: [createproject,%s%s]
    runs-on: ubuntu-latest
    if: always()
    steps:
      - name: Checkout
        uses: actions/checkout@v3
      - name: Set up Go
        uses: actions/setup-go@v3
        with:
          go-version-file: 'go.mod'
          check-latest: true
          cache: true
      - name: Test ${{ matrix.name }} resource
        run: |
          export ACC_TEST_PROJECT_ID=${{needs.createproject.outputs.projectID}}
          if [[ -z "${ACC_TEST_PROJECT_ID}" || "${ACC_TEST_PROJECT_ID}" == "NULL" || "${ACC_TEST_PROJECT_ID}" == "null" ]]; then
            exit 1;
          fi;
          make ci-testacc TEST="./${{ matrix.path }}/..." ACC_TEST_BILLING_REF="${{ secrets.ACC_TEST_BILLING_REF }}" ACC_TEST_USER_EMAIL="${{ secrets.ACC_TEST_USER_EMAIL }}" STACKIT_SERVICE_ACCOUNT_TOKEN="${{ secrets.STACKIT_SERVICE_ACCOUNT_TOKEN }}" STACKIT_SERVICE_ACCOUNT_EMAIL="${{ secrets.STACKIT_SERVICE_ACCOUNT_EMAIL }}"
      - name: Save results
        if: always()
        uses: actions/upload-artifact@v3
        with:
          path: .github/files/analyze-test-output/result/*.json
`, prefix, id, strings.Join(names, ","), incl, previousPrefix, id)
	}

	// handle non restricted matrix
	// collect names
	collectedNames := []string{}
	for _, names := range sorted {
		if len(names) != 1 {
			continue
		}
		collectedNames = append(collectedNames, names...)
	}
	sort.Strings(collectedNames)

	incl := ""
	for _, n := range collectedNames {
		incl = incl + fmt.Sprintf(`
        - name: %s
          path: %s
`, n, keyAndPathMap[n])
	}

	s = s + fmt.Sprintf(`
  resources:
    strategy:
      fail-fast: false
      matrix:
        name: [%s]
        include:
%s
    name: ${{ matrix.name }} resource
    needs: [createproject,datasources]
    runs-on: ubuntu-latest
    if: always()
    steps:
      - name: Checkout
        uses: actions/checkout@v3
      - name: Set up Go
        uses: actions/setup-go@v3
        with:
          go-version-file: 'go.mod'
          check-latest: true
          cache: true
      - name: Test ${{ matrix.name }} resource
        run: |
          export ACC_TEST_PROJECT_ID=${{needs.createproject.outputs.projectID}}
          if [[ -z "${ACC_TEST_PROJECT_ID}" || "${ACC_TEST_PROJECT_ID}" == "NULL" || "${ACC_TEST_PROJECT_ID}" == "null" ]]; then
            exit 1;
          fi;
          make ci-testacc TEST="./${{ matrix.path }}/..." ACC_TEST_BILLING_REF="${{ secrets.ACC_TEST_BILLING_REF }}" ACC_TEST_USER_EMAIL="${{ secrets.ACC_TEST_USER_EMAIL }}" STACKIT_SERVICE_ACCOUNT_TOKEN="${{ secrets.STACKIT_SERVICE_ACCOUNT_TOKEN }}" STACKIT_SERVICE_ACCOUNT_EMAIL="${{ secrets.STACKIT_SERVICE_ACCOUNT_EMAIL }}"
      - name: Save results
        if: always()
        uses: actions/upload-artifact@v3
        with:
          path: .github/files/analyze-test-output/result/*.json
`, strings.Join(collectedNames, ","), incl)

	return s, "[createproject," + strings.Join(nextNeeds, ",") + "]"

}
