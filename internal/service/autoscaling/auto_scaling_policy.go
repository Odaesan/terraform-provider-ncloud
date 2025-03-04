package autoscaling

import (
	"github.com/NaverCloudPlatform/ncloud-sdk-go-v2/ncloud"
	"github.com/NaverCloudPlatform/ncloud-sdk-go-v2/services/autoscaling"
	"github.com/NaverCloudPlatform/ncloud-sdk-go-v2/services/vautoscaling"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
	"regexp"

	. "github.com/terraform-providers/terraform-provider-ncloud/internal/common"
	"github.com/terraform-providers/terraform-provider-ncloud/internal/conn"
	. "github.com/terraform-providers/terraform-provider-ncloud/internal/verify"
)

func ResourceNcloudAutoScalingPolicy() *schema.Resource {
	return &schema.Resource{
		Create: resourceNcloudAutoScalingPolicyCreate,
		Read:   resourceNcloudAutoScalingPolicyRead,
		Update: resourceNcloudAutoScalingPolicyUpdate,
		Delete: resourceNcloudAutoScalingPolicyDelete,
		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},
		Schema: map[string]*schema.Schema{
			"name": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
				ValidateDiagFunc: ToDiagFunc(validation.All(
					validation.StringLenBetween(1, 255),
					validation.StringMatch(regexp.MustCompile(`^[a-z]+[a-z0-9-]+[a-z0-9]$`), "Allows only lowercase letters(a-z), numbers, hyphen (-). Must start with an alphabetic character, must end with an English letter or number"))),
			},
			"adjustment_type_code": {
				Type:             schema.TypeString,
				Required:         true,
				ValidateDiagFunc: ToDiagFunc(validation.StringInSlice([]string{"CHANG", "EXACT", "PRCNT"}, false)),
			},
			"scaling_adjustment": {
				Type:             schema.TypeInt,
				Required:         true,
				ValidateDiagFunc: ToDiagFunc(validation.IntBetween(-2147483648, 2147483647)),
			},
			"cooldown": {
				Type:             schema.TypeInt,
				Optional:         true,
				Computed:         true,
				ValidateDiagFunc: ToDiagFunc(validation.IntBetween(0, 2147483647)),
			},
			"min_adjustment_step": {
				Type:             schema.TypeInt,
				Optional:         true,
				ValidateDiagFunc: ToDiagFunc(validation.IntBetween(1, 2147483647)),
			},
			"auto_scaling_group_no": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
		},
	}
}

func resourceNcloudAutoScalingPolicyCreate(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*conn.ProviderConfig)

	autoscaling_group_no, id, err := createAutoScalingPolicy(d, config)
	if err != nil {
		return err
	}

	d.SetId(ncloud.StringValue(id))
	d.Set("auto_scaling_group_no", autoscaling_group_no)
	return resourceNcloudAutoScalingPolicyRead(d, meta)
}

func createAutoScalingPolicy(d *schema.ResourceData, config *conn.ProviderConfig) (*string, *string, error) {
	if config.SupportVPC {
		return createVpcAutoScalingPolicy(d, config)
	} else {
		return createClassicAutoScalingPolicy(d, config)
	}
}

func createVpcAutoScalingPolicy(d *schema.ResourceData, config *conn.ProviderConfig) (*string, *string, error) {
	reqParams := &vautoscaling.PutScalingPolicyRequest{
		RegionCode: &config.RegionCode,
		// Required
		AdjustmentTypeCode: ncloud.String(d.Get("adjustment_type_code").(string)),
		ScalingAdjustment:  ncloud.Int32(int32(d.Get("scaling_adjustment").(int))),
		AutoScalingGroupNo: ncloud.String(d.Get("auto_scaling_group_no").(string)),
		PolicyName:         ncloud.String(d.Get("name").(string)),
		// Optional
		MinAdjustmentStep: Int32PtrOrNil(d.GetOk("min_adjustment_step")),
		CoolDown:          Int32PtrOrNil(d.GetOk("cooldown")),
	}
	resp, err := config.Client.Vautoscaling.V2Api.PutScalingPolicy(reqParams)
	if err != nil {
		return nil, nil, err
	}

	policy := resp.ScalingPolicyList[0]
	return policy.AutoScalingGroupNo, policy.PolicyNo, nil
}

func createClassicAutoScalingPolicy(d *schema.ResourceData, config *conn.ProviderConfig) (*string, *string, error) {
	no := d.Get("auto_scaling_group_no").(string)
	name := ncloud.String(d.Get("name").(string))
	asg, err := getClassicAutoScalingGroup(config, no)
	if err != nil {
		return nil, nil, err
	}
	reqParams := &autoscaling.PutScalingPolicyRequest{
		// Required
		AdjustmentTypeCode:   ncloud.String(d.Get("adjustment_type_code").(string)),
		ScalingAdjustment:    ncloud.Int32(int32(d.Get("scaling_adjustment").(int))),
		AutoScalingGroupName: asg.AutoScalingGroupName,
		PolicyName:           name,
		// Optional
		MinAdjustmentStep: Int32PtrOrNil(d.GetOk("min_adjustment_step")),
		Cooldown:          Int32PtrOrNil(d.GetOk("cooldown")),
	}

	if _, err := config.Client.Autoscaling.V2Api.PutScalingPolicy(reqParams); err != nil {
		return nil, nil, err
	}

	return ncloud.String(no), name, nil
}

func resourceNcloudAutoScalingPolicyRead(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*conn.ProviderConfig)
	policy, err := GetAutoScalingPolicy(config, d.Id(), d.Get("auto_scaling_group_no").(string))
	if err != nil {
		return err
	}

	if policy == nil {
		d.SetId("")
		return nil
	}

	policyMap := ConvertToMap(policy)
	SetSingularResourceDataFromMapSchema(ResourceNcloudAutoScalingPolicy(), d, policyMap)
	return nil
}

func GetAutoScalingPolicy(config *conn.ProviderConfig, id string, autoScalingGroupNo string) (*AutoScalingPolicy, error) {
	if config.SupportVPC {
		return getVpcAutoScalingPolicy(config, id, autoScalingGroupNo)
	} else {
		return getClassicAutoScalingPolicy(config, id, autoScalingGroupNo)
	}
}

func getVpcAutoScalingPolicy(config *conn.ProviderConfig, id string, autoScalingGroupNo string) (*AutoScalingPolicy, error) {
	reqParams := &vautoscaling.GetAutoScalingPolicyListRequest{
		RegionCode:         &config.RegionCode,
		AutoScalingGroupNo: ncloud.String(autoScalingGroupNo),
		PolicyNoList:       []*string{ncloud.String(id)},
	}
	resp, err := config.Client.Vautoscaling.V2Api.GetAutoScalingPolicyList(reqParams)
	if err != nil {
		return nil, err
	}

	p := resp.ScalingPolicyList[0]
	return &AutoScalingPolicy{
		AutoScalingPolicyNo:   p.PolicyNo,
		AutoScalingPolicyName: p.PolicyName,
		AutoScalingGroupNo:    p.AutoScalingGroupNo,
		AdjustmentTypeCode:    p.AdjustmentType.Code,
		ScalingAdjustment:     p.ScalingAdjustment,
		Cooldown:              p.CoolDown,
		MinAdjustmentStep:     p.MinAdjustmentStep,
	}, nil

}

func getClassicAutoScalingPolicy(config *conn.ProviderConfig, id string, autoScalingGroupNo string) (*AutoScalingPolicy, error) {
	asg, err := getClassicAutoScalingGroup(config, autoScalingGroupNo)
	if err != nil {
		return nil, err
	}

	reqParams := &autoscaling.GetAutoScalingPolicyListRequest{
		PolicyNameList:       []*string{ncloud.String(id)},
		AutoScalingGroupName: asg.AutoScalingGroupName,
	}
	resp, err := config.Client.Autoscaling.V2Api.GetAutoScalingPolicyList(reqParams)
	if err != nil {
		return nil, err
	}
	if len(resp.ScalingPolicyList) < 1 {
		return nil, nil
	}

	p := resp.ScalingPolicyList[0]
	return &AutoScalingPolicy{
		AutoScalingPolicyName: p.PolicyName,
		AdjustmentTypeCode:    p.AdjustmentType.Code,
		ScalingAdjustment:     p.ScalingAdjustment,
		Cooldown:              p.Cooldown,
		MinAdjustmentStep:     p.MinAdjustmentStep,
		AutoScalingGroupNo:    asg.AutoScalingGroupNo,
	}, nil
}

func resourceNcloudAutoScalingPolicyUpdate(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*conn.ProviderConfig)
	_, _, err := createAutoScalingPolicy(d, config)
	if err != nil {
		return err
	}
	return resourceNcloudAutoScalingPolicyRead(d, meta)
}

func resourceNcloudAutoScalingPolicyDelete(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*conn.ProviderConfig)
	if err := deleteAutoScalingPolicy(config, d.Id(), d.Get("auto_scaling_group_no").(string)); err != nil {
		return err
	}
	return nil
}

func deleteAutoScalingPolicy(config *conn.ProviderConfig, id string, autoScalingGroupNo string) error {
	if config.SupportVPC {
		return deleteVpcAutoScalingPolicy(config, id, autoScalingGroupNo)
	} else {
		return deleteClassicAutoScalingPolicy(config, id, autoScalingGroupNo)
	}
}

func deleteVpcAutoScalingPolicy(config *conn.ProviderConfig, id string, autoScalingGroupNo string) error {
	p, err := getVpcAutoScalingPolicy(config, id, autoScalingGroupNo)
	if err != nil {
		return err
	}
	reqParams := &vautoscaling.DeleteScalingPolicyRequest{
		RegionCode:         &config.RegionCode,
		AutoScalingGroupNo: p.AutoScalingGroupNo,
		PolicyNo:           p.AutoScalingPolicyNo,
	}

	if _, err := config.Client.Vautoscaling.V2Api.DeleteScalingPolicy(reqParams); err != nil {
		return err
	}
	return nil
}

func deleteClassicAutoScalingPolicy(config *conn.ProviderConfig, id string, autoScalingGroupNo string) error {
	asg, err := getClassicAutoScalingGroup(config, autoScalingGroupNo)
	if err != nil {
		return err
	}
	reqParams := &autoscaling.DeletePolicyRequest{
		AutoScalingGroupName: asg.AutoScalingGroupName,
		PolicyName:           ncloud.String(id),
	}
	if _, err := config.Client.Autoscaling.V2Api.DeletePolicy(reqParams); err != nil {
		return err
	}
	return nil
}
