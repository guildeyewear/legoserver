# Data Design for GUILD REST server

## Introduction
This document is intended as a point-in-time snapshot of the thinking behind the data model and technology choices for our REST API server.  It is documentation intended to help future interested people understand why things are the way they are.

It is not intended as a reference or standard.  The actual implementation may diverge.  In a perfect world those divergences will be noted in this document, but no guarantees. 

## Context
GUILD is currently working on two major initiatives, a retail and an online shopping experience.

1. The retail experiment consists of launching a traditional collection of glasses in local stores. Glasses will be customized insofar as they can be scaled to individual customers' head sizes, and the noses can be cut to shape. Colors may also be customizable.
2. The online experiment consists of what we're calling a "lego" site, where online users can create a distinct pair of glasses by combining off-the-shelf elements into something personal. 

The previous online experiments centred on website users designing their own glasses from scratch using web-based drawing tools focused on b-splines. While definitely a Technical Triumph, the customer acceptance was less than stellar and so we are trying the two above approaches. The data requirements are distinct enough to warrant taking a fresh look at the model.

## Entities

1. Accounts: Accounts are the retail establishments that are carrying our glasses. GUILD eyewear itself will be a special case of an account, selling glasses through the online store.
2. AccountUsers: Users who can sign in and perform actions on behalf of accounts.
3. Customers: People who actually wear the glasses.  May not actually interact directly with the system, but only through optomotrists (Accounts)
4. Designs: A design represents a specific frame design, including sizes and colors.
5. Materials: The various colors and thicknesses of plastic used to create designs. Each design references a set of acceptable materials for the fronts and temples.
6. Orders: Glasses that need to be made. Orders reference a particular design, which they might modify via a scale and vertical offset. Orders also specify which material is used for the front and temple.

In general the system entities are read more frequently than they are written. The exception are designs while they are in process of being designed, during which time the geometry is frequently written.  This is a low volume transaction however.

## Datastore

We chose mongodb as the datastore because the read-heavy nature of the system lends itself well to a document database. The economic data (accounts, accountusers, etc) will be modeled much as a relational database would be, with each entity in different containers.  However the designs themselves will be modelled as much as possible as documents with properties embedded to enable fast reads.

The initial version contains the following collections.

Accounts represent stores that are selling GUILD eyewear frames. The online store can be thought of as an internal "account". Accounts have access to groups of designs (called collections) that they can sell.  They may be entitled to a discount on some collections.  Each account may have several AccountUsers who are entitled to log in and sell glasses for the account.

```javascript
Accounts {
    _id: bson.ObjectID,
    name: string, // The human-readable name of the account
    locations: [ // Physical location of stores, if any
        {
            address1: string,
            address2: string,
            city: string,
            province: string,
            postalcode: string,
            country: string
        },
    ],
    primaryContact: { // Client-side owner of this account
        name: string,
        phone: string,
        position: string,
    },
    collections: [ // Collections this account can sell
        string, string, ... // Collection names stored in designs
    ],
    discount: {
        string: float32, // Map of collection name to discount for that collection
    },
}
```

AccountUsers are usually employees of the optical stores selling GUILD glasses.  They may log into the various applications (iPad application, web app) for that Account using unique creditials.  That allows us to tie activies within an account to a particual employee and also gives the store the ability to add and remove access for employees.

```javascript
AccountUser {
    _id: string, // Unique id for the user, probably an email address
    account: bson.ObjectId, // ----> Account document
    firstname: string,
    familyname: string,
    type: int16, // 0 == sales, 1 == admin
    pwSalt: string,
    pwHash: string, // Secure hash of password + application salt + user salt
}
```

Customers are people who actually wear the glasses. Their primary attribute is their sizing information, which is used to manufacture the glasses for a custom fit. They might have their own login information (login id and password), for example for use with the website, or they might not, for example if they are a client of one of the optical stores.

```javascript
Customer {
    _id: bson.ObjectId,
    email: string, // Used for a login scenario
    pwSalt: string,
    pwHash: string, // Secure hash of password + application salt + user salt
    firstname: string,
    familyname: string
    sizeInfo: { // All measurements in 1/100 millimeter to allow ints to be used but 0.01mm resolution
        pd: int16, 
        splay: int16, // degrees
        ridge: int16, // degrees
        noseHeight: int16, 
        noseRad: int16, 
        templeLength: int16, 
        earHeight: int16, 
        templeWidth: int16, 
        faceWidth: int16, 
    },
}
```

Designs are the primary entity of the system, representing a fully-specified frame design. Customizations are specified in the individual orders of the designs and represent either parametric modifications to the design or concrete choices among possibilities specified in the design. Designs are grouped by collection, which is contained as an array within the design itself.

```javascript
Design {
    _id: bson.ObjectId,
    name: string,
    front: { // The main front part of the frames
        outerCurve: [
            [int16, int16], ... // Control points in 1/100 mm of the left outer contour b-spline. Right contour is a mirror.
        ],
        lens: [
            [int16, int16], ... // Control points in 1/100 mm of the left lens hole.  Right lens hole is mirrored.
        ],
        holes: [ // Any through-holes cut into the front on the left side.  Right side is mirrored.
            [
                [int16, int16]... // Polyline segments of this hole in 1/100 mm
            ], ... // other holes
        ],
        materials: [  // the color/pattern/thickness acceptable
            bson.ObjectID,  ----> Materials
        ]
        engraving: [
            {
                depth: int16, // in 1/100 mm
                angle: int16, // included angle of engraving bit
                paths: [
                    [
                        [int16, int16]... // polyline segments in 1/100 mm
                    ]... // any other paths using this bit and depth
                ],
            }, ... // Any other depths and angles
       ] 
    }
    temple: { // The arms of the frames
        materials: [  // the color/pattern/thickness acceptable
            bson.ObjectID,  ----> Materials
        ]
        contour: [
            [int16, int16], ... // Control points in 1/100 mm of the left temple contour b-spline. Right contour is a mirror.
        ],
        engraving: [
            {
                depth: int16, // in 1/100 mm
                angle: int16, // included angle of engraving bit
                paths: [
                    [
                        [int16, int16]... // polyline segments in 1/100 mm
                    ]... // any other paths using this bit and depth
                ],
            }, ... // Any other depths and angles
        ], 
        templeWidth: int16, // Separation of the temples in 1/100 mm
        templeHeight: int16, // Location of the temple in the Y axis
    },
    collections: [string, string...], // The collections this design belongs to 
    designer: string, // The designer of this frame ----> AccountUser
}
```

Materials represent the raw material - at this point various kinds of cellulose acetate - that the glasses can be made from. 

```javascript
Materials {
    _id: bson.ObjectID,
    name: string,
    topColor: string, // Hexadecimal color representation
    bottomColor: string, // Hexadecimal color representation
    topOpacity: float16, // 0-1 opacity
    bottomOpacity: float16, // 0-1 opacity
    topPattern: byte[], // PNG of material such as tortoiseshell
    bottomPattern: byte[], // PNG of material such as tortoiseshell
    topManufacturersCode: string, // Mazz code for material
    bottomManufacturersCode: string, // Mazz code for material
    topThickness: int16, // Thickness in 1/100 mm
    bottomThickness: int16, // Thickness in 1/100 mm
    inStock: int32, // Blanks currently available
    photoUrls: [ // URLs of photographic examples of glasses made with this material
        string, string... // Encoded URLs
    ],
}
```

Orders are a customer request for a pair of glasses.

```javascript
Orders {
    _id: bson.ObjectID,
    account: bson.ObjectID, -----> Accounts
    customer: bson.ObjectID, -----> Customers
    accountUser: string, -----> AccountUsers
    front_material: bson.ObjectId, -----> Materials
    temple_material: bson.ObjectId, -----> Materials
    scale: float16, // Amount to scale design larger or smaller.
    y_pos: int16, // Y position adjustment to fit on person, in 1/100 mm
    left_temple_engrave: string, // engraving on left temple
    right_temple_engrave: string, // engraving on right temple
    invoice: {
        type: string, // direct, credit, etc
        invoice_date: time, // When account invoiced
        status: string, // not_due, due, overdue, paid, etc.
        closedOn: time, // When the invoice was closed
    } 
}
```
