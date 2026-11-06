package gui

// TagRows は、タグの一覧を候補の順で返す。
func (s *Service) TagRows() ([]string, error) {
	if s.startErr != nil {
		return nil, s.startErr
	}
	tags, err := s.app().Tags()
	if err != nil {
		return nil, err
	}
	return append([]string{}, tags...), nil
}

// SaveTags はタグの一覧を書く。base は読んだときの一覧。
func (s *Service) SaveTags(base, tags []string) error {
	if s.startErr != nil {
		return s.startErr
	}
	return s.app().SaveTags(base, tags)
}
