package trienode

func (set *MergedNodeSet) Regroup() ([]*MergedNodeSet, *MergedNodeSet, []bool) {
	regrouped := make([]*MergedNodeSet, 17)
	for i := range regrouped { // break into 16 shards
		regrouped[i] = NewMergedNodeSet()
		for owner := range set.Sets {
			regrouped[i].Sets[owner] = NewNodeSet(owner)
		}
	}

	shards := make([]bool, 16)
	for i := 0; i < len(regrouped); i++ {
		for owner, v := range set.Sets {
			for k, v := range v.Nodes {
				if len(k) > 0 {
					shards[k[0]] = true
					// fmt.Println(k[0])
					regrouped[k[0]].Sets[owner].Nodes[k] = v
				} else {
					// fmt.Println(k)
					regrouped[16].Sets[owner].Nodes[k] = v
				}
			}
		}
	}
	return regrouped[0:16], regrouped[16], shards
}

type MergedNodeSets []*MergedNodeSet

func (nodeset MergedNodeSets) Count() int {
	total := 0
	for i := 0; i < len(nodeset); i++ {
		for _, v := range nodeset[i].Sets {
			total += len(v.Nodes)
		}
	}
	return total
}
