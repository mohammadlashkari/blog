package content

type Index struct {
	Posts           []*Post
	PostsBySlug     map[string]*Post
	PostsByTag      map[string][]*Post
	PostsByLanguage map[Language][]*Post
	Tags            []string
}

func (c *Content) buildIndex() (*Index, error) {
	postsBySlug, err := c.fsPosts(c.postsPath)
	if err != nil {
		return nil, err
	}

	i := &Index{
		PostsBySlug:     postsBySlug,
		PostsByTag:      make(map[string][]*Post),
		PostsByLanguage: make(map[Language][]*Post),
		Tags:            make([]string, 0),
	}

	seenTags := make(map[string]struct{})
	for _, p := range postsBySlug {
		i.Posts = append(i.Posts, p)
		i.PostsByLanguage[p.Language] = append(i.PostsByLanguage[p.Language], p)
		for _, tag := range p.Tags {
			if _, ok := seenTags[tag]; !ok {
				i.Tags = append(i.Tags, tag)
			}
			i.PostsByTag[tag] = append(i.PostsByTag[tag], p)
		}
	}

	return i, nil
}
