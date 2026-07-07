package turbopuffer

import (
	"context"
	"sort"

	tpuf "github.com/turbopuffer/turbopuffer-go/v2"
	"github.com/turbot/steampipe-plugin-sdk/v5/grpc/proto"
	"github.com/turbot/steampipe-plugin-sdk/v5/plugin"
	"github.com/turbot/steampipe-plugin-sdk/v5/plugin/transform"
)

// attributeRow is one flattened namespace schema attribute.
type attributeRow struct {
	Namespace         string
	Region            string
	Name              string
	Type              string
	Filterable        bool
	FullTextSearch    bool
	FullTextConfig    interface{}
	Regex             bool
	Glob              bool
	Fuzzy             bool
	VectorIndex       bool
	SparseVectorIndex bool
}

func tableTurbopufferNamespaceAttribute(_ context.Context) *plugin.Table {
	return &plugin.Table{
		Name:        "turbopuffer_namespace_attribute",
		Description: "One row per attribute in each namespace schema: type, filterability, full-text/regex/glob/fuzzy indexing, auto-embedding and vector index flags.",
		List: &plugin.ListConfig{
			ParentHydrate: listNamespaces,
			Hydrate:       listNamespaceAttributes,
			KeyColumns: []*plugin.KeyColumn{
				{Name: "namespace", Require: plugin.Optional},
			},
		},
		GetMatrixItemFunc: regionMatrix,
		Columns: []*plugin.Column{
			{Name: "namespace", Type: proto.ColumnType_STRING, Transform: transform.FromField("Namespace"), Description: "Namespace ID."},
			{Name: "filterable", Type: proto.ColumnType_BOOL, Transform: transform.FromField("Filterable"), Description: "Whether the attribute can be used in filters. ACL attributes MUST be filterable to enforce authorization."},
			{Name: "full_text_config", Type: proto.ColumnType_JSON, Transform: transform.FromField("FullTextConfig"), Description: "Full-text search configuration, if enabled."},
			{Name: "full_text_search", Type: proto.ColumnType_BOOL, Transform: transform.FromField("FullTextSearch"), Description: "Whether BM25 full-text search is enabled on the attribute."},
			{Name: "fuzzy", Type: proto.ColumnType_BOOL, Transform: transform.FromField("Fuzzy"), Description: "Whether fuzzy filters are enabled on the attribute."},
			{Name: "glob", Type: proto.ColumnType_BOOL, Transform: transform.FromField("Glob"), Description: "Whether glob filters are enabled on the attribute."},
			{Name: "name", Type: proto.ColumnType_STRING, Transform: transform.FromField("Name"), Description: "Attribute name."},
			{Name: "regex", Type: proto.ColumnType_BOOL, Transform: transform.FromField("Regex"), Description: "Whether regex filters are enabled on the attribute."},
			{Name: "region", Type: proto.ColumnType_STRING, Transform: transform.FromField("Region"), Description: "turbopuffer region."},
			{Name: "sparse_vector_index", Type: proto.ColumnType_BOOL, Transform: transform.FromField("SparseVectorIndex"), Description: "Whether a sparse kNN index is configured for the attribute."},
			{Name: "type", Type: proto.ColumnType_STRING, Transform: transform.FromField("Type"), Description: "Attribute type (string, int, uuid, datetime, [DIMS]f32 vector, etc.)."},
			{Name: "vector_index", Type: proto.ColumnType_BOOL, Transform: transform.FromField("VectorIndex"), Description: "Whether an ANN vector index is configured for the attribute."},
			{Name: "akas", Type: proto.ColumnType_JSON, Transform: transform.FromValue().Transform(attributeAkas), Description: "Array of globally unique identifiers (region/namespace/name) for the attribute."},
			{Name: "title", Type: proto.ColumnType_STRING, Transform: transform.FromField("Name"), Description: "Title of the resource."},
		},
	}
}

//// LIST HYDRATE FUNCTIONS

// listNamespaceAttributes flattens the parent namespace's schema.
func listNamespaceAttributes(ctx context.Context, d *plugin.QueryData, h *plugin.HydrateData) (interface{}, error) {
	parent := h.Item.(namespaceRow)

	// Only expand the requested namespace, if qualed.
	if q := d.EqualsQualString("namespace"); q != "" && q != parent.ID {
		return nil, nil
	}

	client, err := getClient(ctx, d, parent.Region)
	if err != nil {
		return nil, err
	}
	nsClient := client.Namespace(parent.ID)
	meta, err := nsClient.Metadata(ctx, tpuf.NamespaceMetadataParams{})
	if err != nil {
		return nil, err
	}

	// Sort for stable output across runs.
	names := make([]string, 0, len(meta.Schema))
	for name := range meta.Schema {
		names = append(names, name)
	}
	sort.Strings(names)

	for _, name := range names {
		attr := meta.Schema[name]
		d.StreamListItem(ctx, attributeRow{
			Namespace:         parent.ID,
			Region:            parent.Region,
			Name:              name,
			Type:              string(attr.Type),
			Filterable:        attr.Filterable,
			FullTextSearch:    attr.JSON.FullTextSearch.Valid(),
			FullTextConfig:    attr.FullTextSearch,
			Regex:             attr.Regex,
			Glob:              attr.Glob,
			Fuzzy:             attr.Fuzzy,
			VectorIndex:       attr.JSON.Ann.Valid(),
			SparseVectorIndex: attr.JSON.SparseKnn.Valid(),
		})
		if d.RowsRemaining(ctx) == 0 {
			return nil, nil
		}
	}
	return nil, nil
}

//// TRANSFORM FUNCTIONS

// attributeAkas builds the akas array: region/namespace/name.
func attributeAkas(_ context.Context, td *transform.TransformData) (interface{}, error) {
	a := td.HydrateItem.(attributeRow)
	return []string{a.Region + "/" + a.Namespace + "/" + a.Name}, nil
}
