package router

import (
	"net/http"

	kubeClientModel "github.com/containerum/kube-client/pkg/model"

	"git.containerum.net/ch/volume-manager/pkg/errors"
	"git.containerum.net/ch/volume-manager/pkg/models"
	"git.containerum.net/ch/volume-manager/pkg/router/middleware"
	"git.containerum.net/ch/volume-manager/pkg/server"
	"github.com/containerum/cherry/adaptors/gonic"
	"github.com/containerum/utils/httputil"
	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"
	"github.com/sirupsen/logrus"
)

type volumeHandlers struct {
	tv   *TranslateValidate
	acts server.VolumeActions
}

func (vh *volumeHandlers) directCreateVolumeHandler(ctx *gin.Context) {
	var req model.DirectVolumeCreateRequest
	if err := ctx.ShouldBindWith(&req, binding.JSON); err != nil {
		ctx.AbortWithStatusJSON(vh.tv.BadRequest(ctx, err))
		return
	}
	if err := vh.acts.DirectCreateVolume(ctx.Request.Context(), ctx.Param("ns_id"), req); err != nil {
		ctx.AbortWithStatusJSON(vh.tv.HandleError(err))
		return
	}

	ctx.Status(http.StatusCreated)
}

func (vh *volumeHandlers) importVolumesHandler(ctx *gin.Context) {
	var req kubeClientModel.VolumesList
	if err := ctx.ShouldBindWith(&req, binding.JSON); err != nil {
		ctx.AbortWithStatusJSON(vh.tv.BadRequest(ctx, err))
		return
	}
	for _, vol := range req.Volumes {
		if err := vh.acts.ImportVolume(ctx.Request.Context(), vol.Namespace, vol); err != nil {
			logrus.Warn(err)
		}
	}

	ctx.Status(http.StatusAccepted)
}

func (vh *volumeHandlers) createVolumeHandler(ctx *gin.Context) {
	var req model.VolumeCreateRequest
	if err := ctx.ShouldBindWith(&req, binding.JSON); err != nil {
		ctx.AbortWithStatusJSON(vh.tv.BadRequest(ctx, err))
		return
	}
	if err := vh.acts.CreateVolume(ctx.Request.Context(), ctx.Param("ns_id"), req); err != nil {
		ctx.AbortWithStatusJSON(vh.tv.HandleError(err))
		return
	}

	ctx.Status(http.StatusCreated)
}

func (vh *volumeHandlers) getVolumeHandler(ctx *gin.Context) {
	ret, err := vh.acts.GetVolume(ctx.Request.Context(), ctx.Param("ns_id"), ctx.Param("label"))
	if err != nil {
		ctx.AbortWithStatusJSON(vh.tv.HandleError(err))
		return
	}

	httputil.MaskForNonAdmin(ctx, &ret)

	ctx.JSON(http.StatusOK, ret)
}

func (vh *volumeHandlers) getNamespaceVolumesHandler(ctx *gin.Context) {
	ret, err := vh.acts.GetNamespaceVolumes(ctx.Request.Context(), ctx.Param("ns_id"))

	if err != nil {
		ctx.AbortWithStatusJSON(vh.tv.HandleError(err))
		return
	}

	for i := range ret.Volumes {
		httputil.MaskForNonAdmin(ctx, &ret.Volumes[i])
	}

	ctx.JSON(http.StatusOK, ret)
}

func (vh *volumeHandlers) getUserVolumesHandler(ctx *gin.Context) {
	ret, err := vh.acts.GetUserVolumes(ctx.Request.Context())

	if err != nil {
		ctx.AbortWithStatusJSON(vh.tv.HandleError(err))
		return
	}

	for i := range ret.Volumes {
		httputil.MaskForNonAdmin(ctx, &ret.Volumes[i])
	}

	ctx.JSON(http.StatusOK, ret)
}

func (vh *volumeHandlers) getAllVolumesHandler(ctx *gin.Context) {
	page, perPage, err := getPaginationParams(ctx.Request.URL.Query())
	if err != nil {
		gonic.Gonic(errors.ErrRequestValidationFailed().AddDetailsErr(err), ctx)
		return
	}

	ret, err := vh.acts.GetAllVolumes(ctx.Request.Context(), page, perPage, getFilters(ctx.Request.URL.Query())...)
	if err != nil {
		ctx.AbortWithStatusJSON(vh.tv.HandleError(err))
		return
	}

	ctx.JSON(http.StatusOK, ret)
}

func (vh *volumeHandlers) deleteVolumeHandler(ctx *gin.Context) {
	if err := vh.acts.DeleteVolume(ctx.Request.Context(), ctx.Param("ns_id"), ctx.Param("label")); err != nil {
		ctx.AbortWithStatusJSON(vh.tv.HandleError(err))
		return
	}

	ctx.Status(http.StatusOK)
}

func (vh *volumeHandlers) deleteAllUserVolumesHandler(ctx *gin.Context) {
	if err := vh.acts.DeleteAllUserVolumes(ctx.Request.Context()); err != nil {
		ctx.AbortWithStatusJSON(vh.tv.HandleError(err))
		return
	}

	ctx.Status(http.StatusOK)
}

func (vh *volumeHandlers) deleteAllNamespaceVolumesHandler(ctx *gin.Context) {
	if err := vh.acts.DeleteAllNamespaceVolumes(ctx.Request.Context(), ctx.Param("ns_id")); err != nil {
		ctx.AbortWithStatusJSON(vh.tv.HandleError(err))
		return
	}

	ctx.Status(http.StatusOK)
}

func (vh *volumeHandlers) resizeVolumeHandler(ctx *gin.Context) {
	var req model.VolumeResizeRequest
	if err := ctx.ShouldBindWith(&req, binding.JSON); err != nil {
		ctx.AbortWithStatusJSON(vh.tv.BadRequest(ctx, err))
		return
	}
	if err := vh.acts.ResizeVolume(ctx.Request.Context(), ctx.Param("ns_id"), ctx.Param("label"), req.TariffID); err != nil {
		ctx.AbortWithStatusJSON(vh.tv.HandleError(err))
		return
	}

	ctx.Status(http.StatusOK)
}

func (vh *volumeHandlers) adminResizeVolumeHandler(ctx *gin.Context) {
	var req model.AdminVolumeResizeRequest
	if err := ctx.ShouldBindWith(&req, binding.JSON); err != nil {
		ctx.AbortWithStatusJSON(vh.tv.BadRequest(ctx, err))
		return
	}
	if err := vh.acts.AdminResizeVolume(ctx.Request.Context(), ctx.Param("ns_id"), ctx.Param("label"), req.Capacity); err != nil {
		ctx.AbortWithStatusJSON(vh.tv.HandleError(err))
		return
	}

	ctx.Status(http.StatusOK)
}

func (r *Router) SetupVolumeHandlers(acts server.VolumeActions) {
	handlers := &volumeHandlers{tv: r.tv, acts: acts}

	group := r.engine.Group("/namespaces/:ns_id/volumes")
	adminGroup := r.engine.Group("/admin/namespaces/:ns_id/volumes", httputil.RequireAdminRole(errors.ErrAdminRequired))

	// swagger:operation POST /limits/namespaces/{ns_id}/volumes Volumes DirectCreateVolume
	//
	// Create Volume using only capacity.
	// Should be chosen first storage, where free space allows to create volume with provided capacity.
	//
	// ---
	// parameters:
	//  - $ref: '#/parameters/UserIDHeader'
	//  - $ref: '#/parameters/UserRoleHeader'
	//  - $ref: '#/parameters/SubstitutedUserID'
	//  - $ref: '#/parameters/NamespaceID'
	//  - name: body
	//    in: body
	//    required: true
	//    schema:
	//      $ref: '#/definitions/DirectVolumeCreateRequest'
	// responses:
	//   '201':
	//     description: volume created
	//   default:
	//     $ref: '#/responses/error'
	r.engine.POST("/limits/namespaces/:ns_id/volumes", middleware.WriteAccess, handlers.directCreateVolumeHandler)

	// swagger:operation POST /namespaces/{ns_id}/volumes Volumes CreateVolume
	//
	// Create Volume for User by Tariff.
	// Should be chosen first storage, where free space allows to create volume with provided capacity.
	//
	// ---
	// parameters:
	//  - $ref: '#/parameters/UserIDHeader'
	//  - $ref: '#/parameters/UserRoleHeader'
	//  - $ref: '#/parameters/SubstitutedUserID'
	//  - $ref: '#/parameters/NamespaceID'
	//  - name: body
	//    in: body
	//    required: true
	//    schema:
	//      $ref: '#/definitions/VolumeCreateRequest'
	// responses:
	//   '201':
	//     description: volume created
	//   default:
	//     $ref: '#/responses/error'
	group.POST("", middleware.WriteAccess, handlers.createVolumeHandler)

	// swagger:operation GET /namespaces/{ns_id}/volumes/{label} Volumes GetVolume
	//
	// Get volume.
	//
	// ---
	// parameters:
	//  - $ref: '#/parameters/UserIDHeader'
	//  - $ref: '#/parameters/UserRoleHeader'
	//  - $ref: '#/parameters/SubstitutedUserID'
	//  - $ref: '#/parameters/NamespaceID'
	//  - name: label
	//    in: path
	//    type: string
	//    required: true
	// responses:
	//   '200':
	//     description: volume response
	//     schema:
	//       $ref: '#/definitions/Volume'
	//   default:
	//     $ref: '#/responses/error'
	group.GET("/:label", middleware.ReadAccess, handlers.getVolumeHandler)

	// swagger:operation GET /namespaces/{ns_id}/volumes Volumes GetNamespaceVolumes
	//
	// Get namespace volumes.
	//
	// ---
	// parameters:
	//  - $ref: '#/parameters/UserIDHeader'
	//  - $ref: '#/parameters/UserRoleHeader'
	//  - $ref: '#/parameters/SubstitutedUserID'
	//  - $ref: '#/parameters/NamespaceID'
	// responses:
	//   '200':
	//     description: volumes response
	//     schema:
	//       type: array
	//       items:
	//         $ref: '#/definitions/Volume'
	//   default:
	//     $ref: '#/responses/error'
	group.GET("", middleware.ReadAccess, handlers.getNamespaceVolumesHandler)

	// swagger:operation GET /volumes Volumes GetUserVolumes
	//
	// Get user volumes.
	//
	// ---
	// parameters:
	//  - $ref: '#/parameters/UserIDHeader'
	//  - $ref: '#/parameters/UserRoleHeader'
	//  - $ref: '#/parameters/SubstitutedUserID'
	// responses:
	//   '200':
	//     description: volumes response
	//     schema:
	//       type: array
	//       items:
	//         $ref: '#/definitions/Volume'
	//   default:
	//     $ref: '#/responses/error'
	r.engine.GET("/volumes", middleware.ReadAccess, handlers.getUserVolumesHandler)

	// swagger:operation GET /admin/volumes Volumes GetAllVolumes
	//
	// Get all volumes (admin only).
	//
	// ---
	// parameters:
	//  - $ref: '#/parameters/UserIDHeader'
	//  - $ref: '#/parameters/UserRoleHeader'
	//  - $ref: '#/parameters/SubstitutedUserID'
	//  - $ref: '#/parameters/Filters'
	//  - $ref: '#/parameters/PageNum'
	//  - $ref: '#/parameters/PerPageLimit'
	// responses:
	//   '200':
	//     description: volumes response
	//     schema:
	//       type: array
	//       items:
	//         $ref: '#/definitions/Volume'
	//   default:
	//     $ref: '#/responses/error'
	r.engine.GET("/admin/volumes", httputil.RequireAdminRole(errors.ErrAdminRequired), handlers.getAllVolumesHandler)

	// swagger:operation DELETE /namespaces/{ns_id}/volumes/{label} Volumes DeleteVolume
	//
	// Delete volume.
	//
	// ---
	// parameters:
	//  - $ref: '#/parameters/UserIDHeader'
	//  - $ref: '#/parameters/UserRoleHeader'
	//  - $ref: '#/parameters/SubstitutedUserID'
	//  - $ref: '#/parameters/NamespaceID'
	//  - name: label
	//    in: path
	//    type: string
	//    required: true
	// responses:
	//   '200':
	//     description: volume deleted
	//   default:
	//     $ref: '#/responses/error'
	group.DELETE("/:label", middleware.DeleteAccess, handlers.deleteVolumeHandler)

	// swagger:operation DELETE /namespaces/{ns_id}/volumes Volumes DeleteAllNamespaceVolumes
	//
	// Delete all namespace volumes.
	//
	// ---
	// parameters:
	//  - $ref: '#/parameters/UserIDHeader'
	//  - $ref: '#/parameters/UserRoleHeader'
	//  - $ref: '#/parameters/SubstitutedUserID'
	//  - $ref: '#/parameters/NamespaceID'
	// responses:
	//   '200':
	//     description: volumes deleted
	//   default:
	//     $ref: '#/responses/error'
	group.DELETE("", middleware.DeleteAccess, handlers.deleteAllNamespaceVolumesHandler)

	// swagger:operation DELETE /volumes Volumes DeleteAllUserVolumes
	//
	// Delete all user volumes.
	//
	// ---
	// parameters:
	//  - $ref: '#/parameters/UserIDHeader'
	//  - $ref: '#/parameters/UserRoleHeader'
	//  - $ref: '#/parameters/SubstitutedUserID'
	// responses:
	//   '200':
	//     description: volumes deleted
	//   default:
	//     $ref: '#/responses/error'
	r.engine.DELETE("/volumes", handlers.deleteAllUserVolumesHandler)

	// swagger:operation PUT /namespaces/{ns_id}/volumes/{label} Volumes ResizeVolume
	//
	// Resize volume.
	//
	// ---
	// parameters:
	//  - $ref: '#/parameters/UserIDHeader'
	//  - $ref: '#/parameters/UserRoleHeader'
	//  - $ref: '#/parameters/SubstitutedUserID'
	//  - $ref: '#/parameters/NamespaceID'
	//  - name: label
	//    in: path
	//    type: string
	//    required: true
	//  - name: body
	//    in: body
	//    required: true
	//    schema:
	//      $ref: '#/definitions/VolumeResizeRequest'
	// responses:
	//   '200':
	//     description: volume resized
	//   default:
	//     $ref: '#/responses/error'
	group.PUT("/:label", middleware.WriteAccess, handlers.resizeVolumeHandler)

	// swagger:operation PUT /admin/namespaces/{ns_id}/volumes/{label} Volumes AdminResizeVolume
	//
	// Resize volume (admins only).
	//
	// ---
	// parameters:
	//  - $ref: '#/parameters/UserIDHeader'
	//  - $ref: '#/parameters/UserRoleHeader'
	//  - $ref: '#/parameters/SubstitutedUserID'
	//  - $ref: '#/parameters/NamespaceID'
	//  - name: label
	//    in: path
	//    type: string
	//    required: true
	//  - name: body
	//    in: body
	//    required: true
	//    schema:
	//      $ref: '#/definitions/AdminVolumeResizeRequest'
	// responses:
	//   '200':
	//     description: volume resized
	//   default:
	//     $ref: '#/responses/error'
	adminGroup.PUT("/:label", handlers.adminResizeVolumeHandler)

	// swagger:operation POST /import/volumes Volumes ImportVolumes
	//
	// Import volumes.
	//
	// ---
	// parameters:
	//  - $ref: '#/parameters/UserIDHeader'
	//  - $ref: '#/parameters/UserRoleHeader'
	//  - name: body
	//    in: body
	//    required: true
	//    schema:
	//      $ref: '#/definitions/VolumesList'
	// responses:
	//   '201':
	//     description: volumes imported
	//   default:
	//     $ref: '#/responses/error'
	r.engine.POST("/import/volumes", handlers.importVolumesHandler)

}
