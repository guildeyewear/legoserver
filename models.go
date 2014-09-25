package main

import (
	"crypto/sha512"
	"encoding/hex"
	"log"
	"time"

	"gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"
)

var (
	mgoSession   *mgo.Session
	databaseName = "guild"
)

const (
	USER_NORMAL        = iota
	USER_ACCOUNT_ADMIN = iota
	USER_SYSTEM_ADMIN  = iota
)

// Model entities for the GUILD web application.
type (
	// Address is a physical address. It is not a collection in the
	// mongodb database, but is an embedded document within people and
	// account documents.
	Address struct {
		Address1   string `bson:"address1,omitempty" json:"address1,omitempty"`
		Address2   string `bson:"address2,omitempty" json:"address2,omitempty"`
		City       string `bson:"city,omitempty" json:"city,omitempty"`
		Province   string `bson:"province,omitempty" json:"province,omitempty"`
		Postalcode string `bson:"postalcode,omitempty" json:"postalcode,omitempty"`
		Country    string `bson:"country,omitempty" json:"country,omitempty"`
	}

	// PersonInfo describes any person relevant to the system.
	// It is not a MongoDB collection but rather an embedded
	// document within AccountUser and Order documents.
	PersonInfo struct {
		Firstname  string   `bson:"firstname,omitempty" json:"firstname,omitempty"`
		Familyname string   `bson:"familyname,omitempty" json:"familyname,omitempty"`
		Email      string   `bson:"email,omitempty" json:"email,omitempty"`
		Phone      string   `bson:"phone,omitempty" json:"phone,omitempty"`
		Position   string   `bson:"position,omitempty" json:"position,omitempty"`
		Address    Address  `bson:"mailing_address,omitempty" json:"mailing_address,omitempty"`
		Billing    Address  `bson:"billing_address,omitempty" json:"billing_address,omitempty"`
		SizeInfo   SizeInfo `bson:"size_info,omitempty" json:"size_info,omitempty"`
	}

	// SizeInfo is the facial geometry of a person.  It
	// is relevant if the PersonInfo that contains it
	// is describing a customer. This is not a MongoDB collection
	// but rather an embedded document within a PersonInfo document.
	SizeInfo struct {
		PD           int16 `bson:"pd,omitempty" json:"pd,omitempty"`
		Splay        int16 `bson:"splayangle,omitempty" json:"splayangle,omitempty"`
		Ridge        int16 `bson:"ridgeangle,omitempty" json:"ridgeangle,omitempty"`
		NoseHeight   int16 `bson:"noseheight,omitempty" json:"noseheight,omitempty"`
		NoseRadius   int16 `bson:"noseradius,omitempty" json:"noseradius,omitempty"`
		TempleLength int16 `bson:"templelength,omitempty" json:"templelength,omitempty"`
		EarHeight    int16 `bson:"earheight,omitempty" json:"earheight,omitempty"`
		TempleWidth  int16 `bson:"templewidth,omitempty" json:"templewidth,omitempty"`
		FaceWidth    int16 `bson:"facewidth,omitempty" json:"facewidth,omitempty"`
	}

	// Account is a customer of GUILD eyewear, usually a optometry store.
	// There may be special kinds of accounts like the website account or
	// a distributor account.  Each account might have multiple locations.
	// Each account also has access to collections of glasses, and a
	// discount specific to that collection.
	// An Account is a MongoDB collection.
	Account struct {
		Id          bson.ObjectId    `bson:"_id,omitempty" json:"-"`
		Name        string           `bson:"name" json:"name"`
		Locations   []Address        `bson:"locations" json:"locations"`
		Contact     string           `bson:"contact_id,omitempty" json:"-"`
		Collections []string         `bson:"collections,omitempty" json:"collections,omitempty"`
		Discount    map[string]int16 `bson:"discounts,omitempty" json:"discounts,omitempty"`
	}

	// AccountUser is a user account that can take actions on
	// behalf of an Account.  For example an Account representing
	// a glasses store might have 4 employees with their own
	// login id and password.  Each employee would be an AccountUser.
	// AccountUser is a MongoDB collection.
	User struct {
		Id        string     `bson:"_id,omitempty" json:"id"`
		Password  string     `bson:"-" json:"-"`
		PwSalt    string     `bson:"pwsalt" json:"-"`
		PwHash    string     `bson:"pwhash" json:"-"`
		AccountId string     `bson:"account_id" json:"-"`
		Person    PersonInfo `bson:"person,omitempty" json:"person,omitempty"`
		Type      int16      `bson:"usertype" json:"usertype"`
		Created   time.Time  `bson:"created" json:"-"`
		Updated   time.Time  `bson:"updated" json:"updated"`
	}

	// Engraving describes any special patterns that might be on a
	// design.  It has an array of paths, which are each an array
	// of XY coordinates (with the coordinates in 1/100 mm.)
	// This is not a MongoDB collection but rather an embedded document
	// within the Temple and Front documents.
	Engraving struct {
		Depth int16        `bson:"depth" json:"depth"`
		Angle int16        `bson:"cutter_angle" json:"cutter_angle"`
		Paths [][][2]int16 `bson:"paths" json:"paths"`
	}

	// Temple describes the arms of the glasses.  The assumption is that
	// both temples are identical, so there is only one document rather than one
	// for left and one for right. The Materials refernece the Materials documents
	// that are acceptable for this particular design: an individual order would
	// choose one of the acceptable materials.
	// Temple is not a MongoDB collection but rather is an embedded document within
	// a Design document.
	Temple struct {
		Contour          [][2]int16      `bson:"contour" json:"contour"`
		Materials        []bson.ObjectId `bson:"materials" json:"-"`
		Engraving        Engraving       `bson:"engraving,omitempty" json:"engraving,omitempty"`
		LeftText         string          `bson:"left_text,omitempty" json:"left_text,omitempty"`
		RightText        string          `bson:"right_text,omitempty" json:"right_text,omitempty"`
		TempleSeparation int16           `bson:"temple_separation" json:"temple_separation"`
		TempleHeight     int16           `bson:"temple_height" json:"temple_height"`
	}
	// Front describes the main front of the glasses.
	// The Materials refernece the Materials documents
	// that are acceptable for this particular design: an individual order would
	// choose one of the acceptable materials. The holes are polyline contours
	// describing cutouts in the design.
	// Front is not a MongoDB collection but rather is an embedded document within
	// a Design document.
	Front struct {
		Outercurve [][2]int16      `bson:"outer_curve" json:"outer_curve"`
		Lens       [][2]int16      `bson:"lens" json:"lens"`
		Holes      [][][2]int16    `bson:"holes,omitempty" json:"holes,omitempty"`
		Engraving  Engraving       `bson:"engraving,omitempty" json:"engraving,omitempty"`
		Materials  []bson.ObjectId `bson:"materials" json:"-"`
	}

	// Design describes a complete frame design, including the geometry, size
	// and acceptable materials.  Design is a MongoDB collection.
	Design struct {
		Id          bson.ObjectId `bson:"_id,omitempty" json:"id"`
		Designer    string        `bson:"designer_accountuser_id" json:"-"`
		Name        string        `bson:"name" json:"name"`
		Front       Front         `bson:"front" json:"front"`
		Temple      Temple        `bson:"temple" json:"temple"`
		Collections []string      `bson:"collections,omitempty" json:"collections,omitempty"`
		Updated     time.Time     `bson:"updated" json:"updated"`
	}

	// Material describes an available plastic blank that a temple or front
	// can be made from. If the material is a lamination then all properties
	// will have a "bottom" variant, otherwise the "top" variant describes the
	// material fully.  The manufacturer's code is the Mazzuccelli product code
	// used for ordering. Stock indicates how many blanks are available.
	Material struct {
		Id                     bson.ObjectId `bson:"_id,omitempty" json:"-"`
		Name                   string        `bson:"name" json:"name"`
		TopThickness           int16         `bson:"top_thickness" json:"top_thickness"`
		TopColor               string        `bson:"top_color" json:"top_color"`
		TopOpacity             int16         `bson:"top_opacity" json:"top_opacity"`
		TopPattern             []byte        `bson:"top_pattern,omitempty" json:"top_pattern"`
		TopManufacturerCode    string        `bson:"top_manufacturer_code" json:"-"`
		BottomThickness        int16         `bson:"bottom_thickness,omitempty" json:"bottom_thickness,omitempty"`
		BottomColor            string        `bson:"bottom_color,omitempty" json:"bottom_color,omitempty"`
		BottomOpacity          int16         `bson:"bottom_opacity,omitempty" json:"bottom_opacity,omitempty"`
		BottomPattern          []byte        `bson:"bottom_pattern,omitempty" json:"bottom_pattern,omitempty"`
		BottomManufacturerCode string        `bson:"bottom_manufacturer_code,omitempty" json:"bottom_manufacturer_code,omitempty"`
		Stock                  int32         `bson:"stock" json:"stock"`
		PhotoUrls              []string      `bson:"photo_urls,omitempty" json:"photo_urls,omitempty"`
	}

	// Order instantiates a design into a concrete frame for a customer. It contains
	// references to the account, the user who entered the order, information about the customer,
	// and various customizations to the design.
	Order struct {
		Id              bson.ObjectId `bson:"_id,omitempty" json:"-"`
		AccountId       bson.ObjectId `bson:"account_id" json:"account_id"`
		CustomerInfo    PersonInfo    `bson:"customer_info" json:"customer_info"`
		UserId          string        `bson:"user_id" json:"user_id"`
		FrontMaterial   bson.ObjectId `bson:"front_material_id" json:"front_material_id"`
		TempleMaterial  bson.ObjectId `bson:"temple_material_id" json:"temple_material_id"`
		Scale           float32       `bson:"scale" json:"scale"`
		YPosition       int16         `bson:"y_position" json:"y_position"`
		LeftTempleText  string        `bson:"left_temple_text" json:"left_temple_text"`
		RightTempleText string        `bson:"right_temple_text" json:"right_temple_text"`
	}

	Invoice struct {
		Id          bson.ObjectId `bson:"_id,omitempty" json:"-"`
		AccountId   bson.ObjectId `bson:"account_id" json:"account_id"`
		Issued      time.Time     `bson:"issued" json:"issued"`
		Status      string        `bson:"status" json:"status"`
		Amount      int32         `bson:"amount" json:"amount"`
		Tax         int32         `bson:"tax" json:"tax"`
		AmountPaid  int32         `bson:"amount_paid" json:"amount_paid"`
		PaymentDate time.Time     `bson:"payment_date" json:"payment_date"`
		Due         time.Time     `bson:"due" json:"due"`
		Orders      []Order       `bson:"orders" json:"orders"`
	}
)

// Account objects
func findAccountById(id string) (err error, a Account) {
	log.Printf("Looking for account with id %v", id)
	withCollection("accounts", func(c *mgo.Collection) {
		err = c.FindId(id).One(&a)
	})
	return
}

func createAccount(acct *Account) (err error) {
	acct.Id = bson.NewObjectId()
	withCollection("accounts", func(c *mgo.Collection) {
		err = c.Insert(acct)
	})
	return
}

// User objects
func findUserById(id string) (u User, err error) {
	withCollection("users", func(c *mgo.Collection) {
		err = c.FindId(id).One(&u)
	})
	return
}

func (u *User) validatePassword(password string) bool {
	log.Printf("Validating password %v", password)
	saltedpw := (u.PwSalt + password)
	hash := sha512.New()
	hash.Write([]byte(saltedpw))
	if hex.EncodeToString(hash.Sum(nil)) == u.PwHash {
		return true
	}
	return false
}

func createUser(user *User) (err error) {
	log.Printf("Trying to create user %v", user)
	withCollection("users", func(c *mgo.Collection) {
		err = c.Insert(user)
	})
	return
}

func insertDesign(design *Design) (err error) {
	log.Printf("Trying to insert design %v", design)
	withCollection("designs", func(c *mgo.Collection) {
		err = c.Insert(design)
	})
	return
}

func getDesignsWithCollection(collection string) (designs []Design, err error) {
	log.Printf("Getting designs inside collection %v", collection)
	withCollection("designs", func(c *mgo.Collection) {
		err = c.Find(bson.M{"collections": collection}).All(&designs)
	})
	return
}

// Utility function for managing Mongodb sessions
func getMongoSession() *mgo.Session {
	if mgoSession == nil {
		var err error
		mgoSession, err = mgo.Dial("localhost")
		if err != nil {
			panic(err)
		}
	}
	return mgoSession.Clone()
}
func withCollection(collection string, s func(*mgo.Collection)) {
	session := getMongoSession()
	defer session.Close()
	c := session.DB(databaseName).C(collection)
	s(c)
}
