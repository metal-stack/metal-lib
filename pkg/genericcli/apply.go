package genericcli

import "fmt"

func (a *GenericCLI[C, U, R]) ApplyFromFile(generic Generic[C, U, R], from string) ([]R, error) {
	mc := MultiDocumentYAML[C]{
		fs: a.fs,
	}

	docs, err := mc.ReadAll(from)
	if err != nil {
		return nil, err
	}

	result := []R{}
	mu := MultiDocumentYAML[U]{
		fs: a.fs,
	}

	for index := range docs {
		createDoc := docs[index]

		created, err := generic.Create(createDoc)
		if err != nil {
			return nil, fmt.Errorf("error creating entity: %w", err)
		}

		if created != nil {
			result = append(result, *created)
			continue
		}

		updateDoc, err := mu.ReadIndex(from, index)
		if err != nil {
			return nil, err
		}

		updated, err := generic.Update(updateDoc)
		if err != nil {
			return nil, fmt.Errorf("error updating entity: %w", err)
		}

		result = append(result, updated)
	}

	return result, nil
}
