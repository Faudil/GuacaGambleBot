from src.items.Item import Item, ItemType, ItemRarity


class ForgetPotion(Item):
    def __init__(self):
        super().__init__(
            name="Potion d'oubli",
            price=2500,
            description="Réinitialise ton familier. Reset son estomac et le remet niveau 10.",
            item_type=ItemType.consumable,
            rarity=ItemRarity.common
        )

    async def use(self, ctx, **kwargs):
        await ctx.send("utilise la commande feed à la place, ton familier redeviendra lvl 10 et oubliera tous ses items")
        return False
