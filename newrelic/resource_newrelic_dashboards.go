package newrelic

import (
	"log"
	"strconv"

	"github.com/hashicorp/terraform/helper/schema"
	"github.com/hashicorp/terraform/helper/validation"
	newrelic "github.com/paultyng/go-newrelic/api"
)

func resourceNewRelicDashboards() *schema.Resource {

	return &schema.Resource{
		Create: resourceNewRelicDashboardCreate,
		Read:   resourceNewRelicDashboardRead,
		Update: resourceNewRelicDashboardUpdate,
		Delete: resourceNewRelicDashboardDelete,
		Importer: &schema.ResourceImporter{
			State: schema.ImportStatePassthrough,
		},
		Schema: map[string]*schema.Schema{
			"id": {
				Type:     schema.TypeInt,
				Optional: true,
			},
			"title": {
				Type:     schema.TypeString,
				Required: true,
			},
			"icon": {
				Type:         schema.TypeString,
				Required:     true,
				ValidateFunc: validation.StringInSlice([]string{"none", "archive", "bar-chart", "line-chart", "bullseye", "user"}, false),
			},
			"visibility": {
				Type:         schema.TypeString,
				Optional:     true,
				ValidateFunc: validation.StringInSlice([]string{"owner", "all"}, false),
			},
			"editable": {
				Type:         schema.TypeString,
				Optional:     true,
				Default:      true,
				ValidateFunc: validation.StringInSlice([]string{"read_only", "editable_by_owner", "editable_by_all"}, false),
			},
			"owner_email": {
				Type:     schema.TypeString,
				Optional: true,
				Default:  true,
			},
			"filter": {
				Type:     schema.TypeMap,
				Optional: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"event_types": {
							Type:     schema.TypeList,
							Optional: true,
						},
						"attributes": {
							Type:     schema.TypeList,
							Optional: true,
						},
					},
				},
			},
			// "metadata": {
			// 	Type:     schema.TypeMap,
			// 	Required: true,
			// 	Elem: &schema.Resource{
			// 		Schema: map[string]*schema.Schema{
			// 			"version": {
			// 				Type:     schema.TypeInt,
			// 				Required: true,
			// 			},
			// 		},
			// 	},
			// },
			"widgets": {
				Type:     schema.TypeList,
				Optional: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"visualization": {
							Type:     schema.TypeString,
							Optional: true,
						},
						"account_id": {
							Type:     schema.TypeInt,
							Optional: true,
						},
						"presentation": {
							Type:     schema.TypeMap,
							Optional: true,
						},
						"layout": {
							Type:     schema.TypeMap,
							Optional: true,
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"width": {
										Type:     schema.TypeInt,
										Optional: true,
									},
									"height": {
										Type:     schema.TypeInt,
										Optional: true,
									},
									"row": {
										Type:     schema.TypeInt,
										Optional: true,
									},
									"column": {
										Type:     schema.TypeInt,
										Optional: true,
									},
								},
							},
						},
						"data": {
							Type:     schema.TypeList,
							Optional: true,
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"nrql": {
										Type:     schema.TypeString,
										Optional: true,
									},
								},
							},
						},
					},
				},
			},
		},
	}
}

func buildDashboardStruct(d *schema.ResourceData) *newrelic.GetDashboardResp {

	widgetSet := d.Get("widgets").([]interface{})
	widgets := make([]newrelic.DashboardWidget, len(widgetSet))

	for i, widgetI := range widgetSet {
		widgetM := widgetI.(map[string]interface{})

		presentationP := widgetM["presentation"].(map[string]interface{})
		presentationWidget := newrelic.WidgetPresentation{
			Title: presentationP["title"].(string),
			Notes: presentationP["notes"].(string),
		}

		// TODO: Why does this crash terraform?
		// layoutP := widgetM["layout"].(map[string]interface{})
		// layoutWidget := newrelic.WidgetLayout{
		// 	Width:  layoutP["width"].(int),
		// 	Height: layoutP["height"].(int),
		// 	Row:    layoutP["row"].(int),
		// 	Column: layoutP["column"].(int),
		// }

		widgetDataSet := widgetM["data"].([]interface{})
		dataWidget := make([]newrelic.WidgetData, len(widgetDataSet))

		for k, widgetDataI := range widgetDataSet {
			widgetDataM := widgetDataI.(map[string]interface{})
			dataWidget[k] = newrelic.WidgetData{
				Nrql: widgetDataM["nrql"].(string),
			}
		}

		widgets[i] = newrelic.DashboardWidget{
			Visualization: widgetM["visualization"].(string),
			Presentation:  presentationWidget,
			Data:          dataWidget,
			// Layout:        layoutWidget,
		}
	}

	dashboard := newrelic.GetDashboardResp{
		Dashboard: newrelic.GetDashboardDetail{
			Title:      d.Get("title").(string),
			Icon:       d.Get("icon").(string),
			Visibility: d.Get("visibility").(string),
			Editable:   d.Get("editable").(string),
			Widgets:    widgets,
			// Metadata: d.Get("metadata").(map[string]interface{}),
		},
	}

	dashboard.Dashboard.Metadata = newrelic.DashboardMetadata{
		Version: 1,
	}

	// dashboard.Dashboard.Filter = newrelic.Filter{
	// 	EventTypes: 1,
	// 	Attributes:
	// }

	log.Printf("[INFO] dashboard struct: %+v", dashboard)
	return &dashboard
}

func readDashboardStruct(dashboard newrelic.GetDashboardDetail, d *schema.ResourceData) error {
	// ids, err := parseIDs(d.Id(), 2)
	// if err != nil {
	// 	return err
	// }

	// policyID := ids[0]

	d.Set("title", dashboard.Title)
	d.Set("icon", dashboard.Icon)
	d.Set("visibility", dashboard.Visibility)
	d.Set("editable", dashboard.Editable)

	return nil
}

func resourceNewRelicDashboardCreate(d *schema.ResourceData, meta interface{}) error {
	log.Printf("[INFO] Creating New Relic dashboard")
	client := meta.(*newrelic.Client)
	condition := buildDashboardStruct(d)

	log.Printf("[INFO] Creating New Relic dashboard %s", condition.Dashboard.Title)

	condition, err := client.CreateDashboard(*condition)
	if err != nil {
		return err
	}

	d.SetId(serializeIDs([]int{condition.Dashboard.ID}))

	return resourceNewRelicDashboardRead(d, meta)
}

func resourceNewRelicDashboardRead(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*newrelic.Client)

	log.Printf("[INFO] Reading New Relic Dashboard %s", d.Id())

	// ids, err := parseIDs(d.Id(), 1)
	// if err != nil {
	// 	return err
	// }
	//
	// // policyID := ids[0]
	// id := ids[0]

	// id = int(d.Id())

	id, _ := strconv.Atoi(d.Id())

	condition, err := client.GetDashboard(id)
	if err != nil {
		if err == newrelic.ErrNotFound {
			d.SetId("")
			return nil
		}

		return err
	}

	log.Printf("[INFO] End of resourceNewRelicDashboardRead")

	return readDashboardStruct(condition, d)
}

func resourceNewRelicDashboardUpdate(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*newrelic.Client)
	condition := buildDashboardStruct(d)

	// ids, err := parseIDs(d.Id(), 1)
	// if err != nil {
	// 	return err
	// }
	//
	// // policyID := ids[0]
	// id := ids[1]

	// condition.PolicyID = policyID
	id, _ := strconv.Atoi(d.Id())
	condition.Dashboard.ID = id

	log.Printf("[INFO] Updating New Relic Synthetics alert condition %d", id)

	_, err := client.UpdateDashboard(id, *condition)
	if err != nil {
		return err
	}

	return resourceNewRelicDashboardRead(d, meta)
}

func resourceNewRelicDashboardDelete(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*newrelic.Client)

	// ids, err := parseIDs(d.Id(), 1)
	// if err != nil {
	// 	return err
	// }
	//
	// // policyID := ids[0]
	// id := ids[1]

	id, _ := strconv.Atoi(d.Id())

	log.Printf("[INFO] Deleting New Relic Synthetics alert condition %d", id)

	if err := client.DeleteDashboard(id); err != nil {
		return err
	}

	d.SetId("")

	return nil
}
