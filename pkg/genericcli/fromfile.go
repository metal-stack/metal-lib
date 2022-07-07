package genericcli

import "fmt"

func (a *GenericCLI[C, U, R]) CreateFromFile(from string) (R, error) {
	var zero R

	mc := MultiDocumentYAML[C]{
		fs: a.fs,
	}

	doc, err := mc.ReadOne(from)
	if err != nil {
		return zero, err
	}

	result, err := a.g.Create(doc)
	if err != nil {
		return zero, fmt.Errorf("error creating entity: %w", err)
	}

	return *result, nil
}

func (a *GenericCLI[C, U, R]) UpdateFromFile(from string) (R, error) {
	var zero R

	mc := MultiDocumentYAML[U]{
		fs: a.fs,
	}

	doc, err := mc.ReadOne(from)
	if err != nil {
		return zero, err
	}

	result, err := a.g.Update(doc)
	if err != nil {
		return zero, fmt.Errorf("error updating entity: %w", err)
	}

	return result, nil
}

func (a *GenericCLI[C, U, R]) ApplyFromFile(from string) ([]R, error) {
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

		created, err := a.g.Create(createDoc)
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

		updated, err := a.g.Update(updateDoc)
		if err != nil {
			return nil, fmt.Errorf("error updating entity: %w", err)
		}

		result = append(result, updated)
	}

	return result, nil
}
