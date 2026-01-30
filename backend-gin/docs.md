# K.I.S.S principle
    ^
    |
 **Keep It Simple Stupid**

 All endpoints are in ``` routes.go``` file

 1.) -> First page is *login page* ```go public.POST("/login", authHandler.Login) ```. Tenants have no         dashbords or accounts to the system at the moment(reason we have few landlords :).
        we are going to make use of *google AOuth2*. landlords/Admins will just have to login via google -Everyone uses google, no doubt
                    -Why this helps:
                                a) password remembering for landlords is difficult; strong passwd I mean
                                b) It will do away with password reset functionality; which is always hectic to most principle
                                c) 

2.) -> Admins logs in, he's able to Create user(landlord) etc

    ```go
    adminRoutes := r.Group("/sudo")
	adminRoutes.Use(
		middleware.AuthMiddleware([]byte(cfg.JWT.Secret)), //sets user in context
		middleware.RequireRole(db, "admin"),               //checks role
	)
	{
		adminRoutes.POST("register", authHandler.Register) //register landlord
		//admin.GET("/users", handlers.ListLandlords)
 		//we'll add more functionality as the app grows(K.I.S.S)
	}
   ```
   **Reqire role** function checks in users table under role column if role matches

3.) -> Landlord logs in:

```go 
landlord := r.Group("/api/v1")
	landlord.Use(middleware.AuthMiddleware([]byte(cfg.JWT.Secret)))
	{
		// Properties
		landlord.POST("/properties/newProperty", handlers.CreateProperty(db))
		landlord.GET("/properties/myProperties", handlers.ListProperties(db))
		landlord.GET("/properties/:propertyId", handlers.GetProperty(db))
		landlord.PATCH("/properties/:updatePropertyByID", handlers.UpdateProperty(db))
		//landlord.DELETE("/properties/:propertyId", controllers.DeleteProperty) // they can't  delete property at the moment,

		// Units (nested under properties)
		landlord.POST("/properties/:propertyId/newUnit", handlers.CreateUnit(db))
		landlord.GET("/properties/:propertyId/units", handlers.GetUnitsByProperty(db))
		//landlord.GET("/units/:unitId", controllers.GetUnit) --jj
	    //landlord.PUT("/units/:unitId", controllers.UpdateUnit)
		//landlord.DELETE("/units/:unitId", controllers.DeleteUnit) --ii

		// Tenants (assigned to units)
		landlord.POST("/units/:unitId/addTenant", handlers.CreateTenant(db))
		landlord.GET("/units/:unitId/tenants", handlers.ListTenants(db))
		landlord.GET("/tenants/:tenantId", handlers.GetTenant(db))
		landlord.PUT("/tenants/:tenantId", handlers.UpdateTenant(db))
		landlord.DELETE("/tenants/:tenantId", handlers.RemoveTenant(db))
	}
```

I've used jwt as http only cookie to set user id in context and not accept user id from request body ie
 ```json "landlord_id: 1"```; frontend does not interact with user id, browser fetches id directly from the server.This avoids CSRF

Tenant pays normaly from his phone, since we have have his payment no in the database, it will automatically reflect as paid and show balance, just like school fee. Customer to Business(C2B) daraja api will apply here

I'll add google auth2.0 last, plus calling my rate limiter function


# MPESA C2B Integration Flow (Landlord / Property System)

This document explains how the system handles **landlord properties, paybills, tills, and MPESA payments**.  

---

## 1️⃣ Landlord Creates a Property and Adds a Paybill

When a landlord creates a property and adds a Paybill/Till:

1. **Function:** `CreatePaybill`  
2. **Actions:**
   - Inserts a new paybill into the `paybills` table.
   - Links the paybill to the landlord.
   - Associates it with a `daraja_apps` record (creating one if none exists for this landlord).
3. **Automatic Actions:**
   - Calls `RegisterDarajaCallbacks`:
     - Uses `GetDarajaAccessToken` to authenticate with Safaricom Daraja.
     - Registers the `ValidationURL` and `ConfirmationURL` for the paybill.
   - Marks `callback_registered = true` in the database once successful.
4. **Secrets Handling:**
   - Consumer key, consumer secret, and passkey are stored **encrypted** in `daraja_apps`.

**Outcome:**  
The Paybill is now **active in Daraja** and ready to receive payments.
Same as Till

---

## 2️⃣ Tenant Makes a Payment

When a tenant pays rent via MPESA:

1. **Daraja C2B Flow:**
   1. MPESA sends a **Validation request** to `/mpesa/validation`.
   2. The handler responds with:

   ```json
   {"ResultCode":0,"ResultDesc":"Accepted"}
   ```

   3. MPESA sends a Confirmation request to `/mpesa/confirmation`.

   4. **Function:** `MpesaPaymentConfirmation`

	**Actions:**
		-Normalizes tenant phone numbers.
		-Detects whether the transaction is a Paybill or Till payment.
		-Finds the tenant associated with the phone number and property.
		-Inserts a transaction into the appropriate table:

##  3️⃣ paybill_transactions or till_transactions.

Updates the tenant’s balance.

Enforces idempotency (duplicate TransIDs are ignored).

**Outcome:**
Payment is processed automatically, tenant balance updated, and the transaction is logged for the landlord’s property.

## 4 Multiple Properties / Shared Tills

	- Multiple properties can share the same Till number:
	- The system identifies the landlord from the till number.
	- Matches tenant by phone number.
	- Applies the payment correctly.
	- Each Paybill is registered individually in Daraja for each property.
	- Shared Till numbers do not require multiple registrations.

## Example Confirmation payload
```json
{
  "BusinessShortCode": "600000",
  "BillRefNumber": "A101",
  "MSISDN": "254708374149",
  "TransAmount": 100,
  "TransID": "TEST123",
  "TransTime": "20240101123045"
}
```
## Normalized Payment stored in DB
```json
{
  "Provider": "MPESA",
  "Channel": "PAYBILL",
  "BusinessID": "600000",
  "AccountRef": "A101",
  "Phone": "254708374149",
  "Amount": 100,
  "Receipt": "TEST123"
}
```

    

    