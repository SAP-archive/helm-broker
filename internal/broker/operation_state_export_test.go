package broker

func NewInstanceStateService(ocg operationCollectionGetter) *instanceStateService {
	return &instanceStateService{
		operationCollectionGetter: ocg,
	}
}

func NewBindStateService(bocg bindOperationCollectionGetter) *bindStateService {
	return &bindStateService{
		bindOperationCollectionGetter: bocg,
	}
}
